package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rmqChanneler interface {
	Channel() (*amqp.Channel, error)
}

type RidePublisher struct {
	rmq    rmqChanneler
	logger *slog.Logger
}

func NewRidePublisher(rmq rmqChanneler, logger *slog.Logger) *RidePublisher {
	return &RidePublisher{rmq: rmq, logger: logger}
}

// PublishStatus publishes ride.status.{ride_id} messages to ride_topic exchange.
func (p *RidePublisher) PublishStatus(ctx context.Context, rideID, status, passengerID string) error {
	ch, err := p.rmq.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}

	msg := map[string]any{
		"ride_id":      rideID,
		"status":       status,
		"passenger_id": passengerID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	routingKey := fmt.Sprintf("ride.status.%s", rideID)
	if err := ch.PublishWithContext(ctx,
		"ride_topic",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	p.logger.Info("ride_status_published",
		"ride_id", rideID, "status", status)
	return nil
}
