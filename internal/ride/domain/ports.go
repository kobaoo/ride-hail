package domain

import (
	"context"
	"time"
)

type RideRepository interface {
	// Create inserts coordinates + ride row; returns ride ID (uuid) and error.
	Create(ctx context.Context, req RideRequest, fare float64, rideNumber string) (string, error)
	CancelRide(ctx context.Context, rideID, passengerID, reason string) (time.Time, error)
}

type Publisher interface {
	PublishStatus(ctx context.Context, rideID, status, passengerID string) error
}

type WebSocketPort interface {
	SendToPassenger(ctx context.Context, passengerID string, msg any) error
}
