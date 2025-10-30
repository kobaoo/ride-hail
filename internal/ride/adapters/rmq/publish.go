package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"ride-hail/internal/common/log"
	"ride-hail/internal/common/rabbitmq"
	"ride-hail/internal/ride/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQPublisher struct {
	rmq *rabbitmq.ManagerMQ
	log *slog.Logger
}

func NewPublisher(r *rabbitmq.ManagerMQ, logger *slog.Logger) domain.Publisher {
	log.InfoX(logger, "publisher_init", "rmq publisher initialized")
	return &RMQPublisher{
		rmq: r,
		log: logger.With("component", "rmq-publisher"),
	}
}

func (p *RMQPublisher) PublishRideRequest(ctx context.Context, payload any, rideType, corrID string) error {
	rk := "ride.request." + rideType
	log.Info(ctx, p.log, "publish_ride_req", "start rk="+rk+" corr_id="+corrID)

	ch, err := p.rmq.Channel()
	if err != nil {
		log.Error(ctx, p.log, "publish_ride_req", "channel unavailable", err)
		return err
	}
	log.Info(ctx, p.log, "publish_ride_req", "channel acquired")

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error(ctx, p.log, "publish_ride_req", "json marshal failed", err)
		return err
	}
	log.Info(ctx, p.log, "publish_ride_req", "payload encoded")

	msg := amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		Timestamp:     time.Now(),
		Body:          body,
		CorrelationId: corrID,
	}

	if err := ch.PublishWithContext(ctx, "ride_topic", rk, false, false, msg); err != nil {
		log.Error(ctx, p.log, "publish_ride_req", "Publish failed", err)
		return err
	}

	log.Info(ctx, p.log, "publish_ride_req", "ok rk="+rk+" corr_id="+corrID)
	return nil
}

func (p *RMQPublisher) PublishRideStatus(ctx context.Context, payload any, status, corrID string) error {
	rk := "ride.status." + status
	log.Info(ctx, p.log, "publish_ride_status", "start rk="+rk+" corr_id="+corrID)

	ch, err := p.rmq.Channel()
	if err != nil {
		log.Error(ctx, p.log, "publish_ride_status", "channel unavailable", err)
		return err
	}
	log.Info(ctx, p.log, "publish_ride_status", "channel acquired")

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error(ctx, p.log, "publish_ride_status", "json marshal failed", err)
		return err
	}
	log.Info(ctx, p.log, "publish_ride_status", "payload encoded")

	msg := amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		Timestamp:     time.Now(),
		Body:          body,
		CorrelationId: corrID,
	}

	if err := ch.PublishWithContext(ctx, "ride_topic", rk, false, false, msg); err != nil {
		log.Error(ctx, p.log, "publish_ride_status", "publish failed", err)
		return err
	}

	log.Info(ctx, p.log, "publish_ride_status", "ok rk="+rk+" corr_id="+corrID)
	return nil
}
