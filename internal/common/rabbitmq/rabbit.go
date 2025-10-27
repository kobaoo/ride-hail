package rabbitmq

import (
	"context"
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
				log.ErrorX(m.log, "rmq_reconnection", "Error with closed Rabbit MQ connection")
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
					
				} else {
					log.ErrorX(m.log, "rmq_reconnection", )
				}
			}
		}
	}
}