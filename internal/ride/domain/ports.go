package domain

import "context"

type RideRepository interface {
	Insert(ctx context.Context, rd *Ride) error
	Cancel(ctx context.Context, rideID string, _ RideStatus) error
}