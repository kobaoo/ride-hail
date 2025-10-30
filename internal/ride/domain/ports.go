package domain

import (
	"context"
)

type RideRepository interface {
	Insert(ctx context.Context, rd *Ride) error
	Cancel(ctx context.Context, rideID string, _ RideStatus) error
}

type Publisher interface {
	PublishRideRequest(ctx context.Context, payload any, rideType string, corrID string) error
	PublishRideStatus(ctx context.Context, payload any, status string, corrID string) error
}

type Consumer interface {
	// Driver responses (driver.response.*) → единый общий consumer, фильтруем по ride_id
	StartDriverResponses(ctx context.Context, handle func(msg DriverResponse) error) error
	// Location fanout → поток локаций для данного сервиса
	StartLocationUpdates(ctx context.Context, handle func(msg LocationUpdate) error) error
}

type RideService interface {
	Validate(r *RideRequest) error 
	CalcFare(r *RideRequest) (float64, float64, int)
	CreateRide(ctx context.Context, fare float64, distance float64, in *RideRequest) (*Ride, error) 
	// GetRide(ctx context.Context, id string) (*Ride, error)
	// UpdateRideStatus(ctx context.Context, id string, status RideStatus) (*Ride, error)
	// EstimateETA(ctx context.Context, id string) (int64, error)
}