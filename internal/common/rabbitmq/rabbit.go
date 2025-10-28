package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"ride-hail/internal/common/config"
	"ride-hail/internal/common/log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ManagerMQ struct {
	Conn     *amqp.Connection
	Chan     *amqp.Channel
	url      string
	log     *slog.Logger
	closedCh chan struct{}
	mu 		 sync.RWMutex
	alive  bool
}

func NewMQ(cfg *config.RMQ, logger *slog.Logger) *ManagerMQ {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/", cfg.User, cfg.Password, cfg.Host, cfg.Port,
	)
	return &ManagerMQ{
		url: url,
		log: logger,
		closedCh: make(chan struct{}),
	}
}

func (m *ManagerMQ) Connect(ctx context.Context) error {
	if err := m.connectOnce(); err != nil {
		return err
	}

	go m.reconnectLoop(ctx)
	return nil
}

func (m *ManagerMQ) connectOnce() error {
	log.InfoX(m.log, "rmq_connect_once", "Connecting to RMQ")

	conn, err := amqp.DialConfig(m.url, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale: "en_US",
		Dial: amqp.DefaultDial(10 * time.Second),
	})

	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = ch.Close()
		return err
	}

	m.mu.Lock()
	m.Conn, m.Chan = conn, ch
	m.alive = true
	m.mu.Unlock()

	log.InfoX(m.log, "rmq_connect_once", "Successfully connected to RMQ")
	return nil
}

func (m *ManagerMQ) reconnectLoop(ctx context.Context) {
	notifyClose := m.Conn.NotifyClose(make(chan *amqp.Error, 1))
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <- ctx.Done():
			m.Chan.Close()
			return
		case err := <-notifyClose:
			m.mu.Lock()
			m.alive = false
			m.mu.Unlock()
			if err != nil {
				log.ErrorX(m.log, "rmq_reconnection", "Error with closed Rabbit MQ connection", err)
			}
			for attempt := 0; ; attempt++ {
				select {
				case <-ctx.Done():
					return 
				case <- ticker.C:
					// reconnection each 4 seconds
				}
				if e := m.connectOnce(); e != nil {
					notifyClose = m.Conn.NotifyClose(make(chan *amqp.Error, 1))
					if e2 := m.DeclareTopology(); e2 != nil {
						log.ErrorX(m.log, "rmq_declare_topology_fail", "Failed to redeclare topology in RMQ", err)
						continue
					}
					break
				} else {
					log.InfoX(m.log, "rmq_reconnection", "Reconnecting to RMQ")
				}
			}	
		}
	}
}

func (m *ManagerMQ) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Chan != nil {
		_ = m.Chan.Close()
	}
	if m.Conn != nil {
		_ = m.Conn.Close()
	}
	if m.alive {
		m.alive = false
	}
	select {
	case <-m.closedCh:
	default:
		close(m.closedCh)
	}
	log.InfoX(m.log, "rmq_closed", "Rabbit MQ closed")
}

func (m *ManagerMQ) Channel() (*amqp.Channel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.alive || m.Chan == nil {
		return nil, errors.New("channel not available")
	}
	return m.Chan, nil
}

func (m *ManagerMQ) DeclareTopology() error {
	ch, err := m.Channel()
	if err != nil {
		return err
	}

	if err := ch.ExchangeDeclare("ride_topic", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("driver_topic", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("location_fanout", "fanout", true, false, false, false, nil); err != nil {
		return err
	}

	type qb struct{ q, ex, key string; isFanout bool }
	for _, b := range []qb{
		{"ride_requests", "ride_topic", "ride.request.*", false},
		{"ride_status", "ride_topic", "ride.status.*", false},
		{"driver_matching", "ride_topic", "ride.request.*", false},
		{"driver_responses", "driver_topic", "driver.response.*", false},
		{"driver_status", "driver_topic", "driver.status.*", false},
		{"location_updates_ride", "location_fanout", "", true}, // fanout игнорит key
	} {
		_, err := ch.QueueDeclare(b.q, true, false, false, false, nil)
		if err != nil {
			return err
		}
		key := b.key
		if b.isFanout {
			key = ""
		}
		if err := ch.QueueBind(b.q, key, b.ex, false, nil); err != nil {
			return err
		}
	}
	return nil
}