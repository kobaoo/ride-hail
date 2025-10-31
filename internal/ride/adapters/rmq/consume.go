package queue

// // internal/ride/queue/consume.go
// package queue

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"log/slog"
// 	"strconv"
// 	"time"

// 	"ride-hail/internal/common/log"
// 	"ride-hail/internal/common/rabbitmq"
// 	"ride-hail/internal/ride/domain"

// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// const (
// 	prefetchDriverResponses = 50
// 	prefetchLocationUpdates = 200
// )

// type RMQConsumer struct {
// 	rmq *rabbitmq.ManagerMQ
// 	log *slog.Logger
// }

// func NewConsumer(r *rabbitmq.ManagerMQ, logger *slog.Logger) domain.Consumer {
// 	log.InfoX(logger, "consumer_init", "rmq consumer initialized")
// 	return &RMQConsumer{
// 		rmq: r,
// 		log: logger.With("component", "rmq-consumer"),
// 	}
// }

// // ---------------- Driver Responses ----------------

// func (c *RMQConsumer) StartDriverResponses(ctx context.Context, handle func(domain.DriverResponse) error) error {
// 	log.InfoX(c.log, "drv_resp_consume_start", "queue=driver_responses prefetch=50")

// 	ch, err := c.rmq.Channel()
// 	if err != nil {
// 		log.ErrorX(c.log, "drv_resp_consume_start", "channel unavailable", err)
// 		return err
// 	}

// 	if err := ch.Qos(prefetchDriverResponses, 0, false); err != nil {
// 		log.ErrorX(c.log, "drv_resp_consume_start", "qos failed", err)
// 		return err
// 	}

// 	deliveries, err := ch.Consume(
// 		"driver_responses",            // queue
// 		"ride-service-driver-responses", // consumer tag
// 		false, // autoAck
// 		false, // exclusive
// 		false, // noLocal
// 		false, // noWait
// 		nil,
// 	)
// 	if err != nil {
// 		log.ErrorX(c.log, "drv_resp_consume_start", "consume failed", err)
// 		return err
// 	}

// 	go func() {
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				log.InfoX(c.log, "drv_resp_consume_stop", "context done")
// 				return
// 			case d, ok := <-deliveries:
// 				if !ok {
// 					log.InfoX(c.log, "drv_resp_consume_stop", "deliveries channel closed")
// 					return
// 				}
// 				c.processDriverResponse(d, handle)
// 			}
// 		}
// 	}()

// 	return nil
// }

// func (c *RMQConsumer) processDriverResponse(d amqp.Delivery, handle func(domain.DriverResponse) error) {
// 	start := time.Now()

// 	var msg domain.DriverResponse
// 	if err := json.Unmarshal(d.Body, &msg); err != nil {
// 		log.ErrorX(c.log, "drv_resp_decode", "json error", err)
// 		_ = d.Nack(false, false)
// 		return
// 	}

// 	if msg.CorrelationID == "" && d.CorrelationId != "" {
// 		msg.CorrelationID = d.CorrelationId
// 	}

// 	log.InfoX(c.log, "drv_resp_rcv",
// 		"ride_id="+msg.RideID+
// 			" driver_id="+msg.DriverID+
// 			" accepted="+boolToStr(msg.Accepted)+
// 			" corr_id="+msg.CorrelationID)

// 	if err := safeHandleDriverResponse(handle, msg); err != nil {
// 		log.ErrorX(c.log, "drv_resp_handle", "handler failed", err)
// 		_ = d.Nack(false, true)
// 		return
// 	}

// 	_ = d.Ack(false)
// 	log.InfoX(c.log, "drv_resp_ack",
// 		"ride_id="+msg.RideID+" took="+time.Since(start).String())
// }

// func safeHandleDriverResponse(h func(domain.DriverResponse) error, m domain.DriverResponse) (err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			err = errors.New("panic in handler")
// 		}
// 	}()
// 	return h(m)
// }

// // ---------------- Location Updates (fanout) ----------------

// func (c *RMQConsumer) StartLocationUpdates(ctx context.Context, handle func(domain.LocationUpdate) error) error {
// 	log.InfoX(c.log, "loc_upd_consume_start", "queue=location_updates_ride prefetch=200")

// 	ch, err := c.rmq.Channel()
// 	if err != nil {
// 		log.ErrorX(c.log, "loc_upd_consume_start", "channel unavailable", err)
// 		return err
// 	}

// 	if err := ch.Qos(prefetchLocationUpdates, 0, false); err != nil {
// 		log.ErrorX(c.log, "loc_upd_consume_start", "qos failed", err)
// 		return err
// 	}

// 	deliveries, err := ch.Consume(
// 		"location_updates_ride",
// 		"ride-service-location-updates",
// 		false, // autoAck
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		log.ErrorX(c.log, "loc_upd_consume_start", "consume failed", err)
// 		return err
// 	}

// 	go func() {
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				log.InfoX(c.log, "loc_upd_consume_stop", "context done")
// 				return
// 			case d, ok := <-deliveries:
// 				if !ok {
// 					log.InfoX(c.log, "loc_upd_consume_stop", "deliveries channel closed")
// 					return
// 				}
// 				c.processLocationUpdate(d, handle)
// 			}
// 		}
// 	}()

// 	return nil
// }

// func (c *RMQConsumer) processLocationUpdate(d amqp.Delivery, handle func(domain.LocationUpdate) error) {
// 	var msg domain.LocationUpdate
// 	if err := json.Unmarshal(d.Body, &msg); err != nil {
// 		log.ErrorX(c.log, "loc_upd_decode", "json error", err)
// 		_ = d.Nack(false, false)
// 		return
// 	}

// 	log.InfoX(c.log, "loc_upd_rcv",
// 		"ride_id="+msg.RideID+
// 			" driver_id="+msg.DriverID+
// 			" lat="+fmtFloat(msg.Location.Lat)+
// 			" lng="+fmtFloat(msg.Location.Lng),)

// 	if err := safeHandleLocationUpdate(handle, msg); err != nil {
// 		log.ErrorX(c.log, "loc_upd_handle", "handler failed", err)
// 		_ = d.Nack(false, true)
// 		return
// 	}

// 	_ = d.Ack(false)
// 	log.InfoX(c.log, "loc_upd_ack", "ride_id="+msg.RideID)
// }

// func safeHandleLocationUpdate(h func(domain.LocationUpdate) error, m domain.LocationUpdate) (err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			err = errors.New("panic in handler")
// 		}
// 	}()
// 	return h(m)
// }

// // ---------------- helpers ----------------

// func boolToStr(b bool) string {
// 	if b {
// 		return "true"
// 	}
// 	return "false"
// }

// func fmtFloat(v float64) string {
// 	// короткий формат для логов
// 	return strconv.FormatFloat(v, 'f', 6, 64)
// }
