package app

import (
	"context"
	"fmt"

	"ride-hail/internal/ride/domain"
)

// AppService coordinates ride creation and notifications.
type AppService struct {
	rideRepo  domain.RideRepository
	publisher domain.Publisher
	wsPort    domain.WebSocketPort
}

func NewAppService(repo domain.RideRepository, pub domain.Publisher, ws domain.WebSocketPort) *AppService {
	return &AppService{rideRepo: repo, publisher: pub, wsPort: ws}
}

func (a *AppService) CreateRide(ctx context.Context, req domain.RideRequest) (*domain.RideResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	fare, dist, dur := req.EstimateFare()

	rideNumber := generateRideNumber() // uses your existing generator in app package

	rideID, err := a.rideRepo.Create(ctx, req, fare, rideNumber)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	if err := a.publisher.PublishStatus(ctx, rideID, "REQUESTED", req.PassengerID); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPublishFailed, err)
	}

	if a.wsPort != nil {
		msg := map[string]any{
			"type":    "ride_update",
			"status":  "REQUESTED",
			"message": "Your ride request has been sent",
			"ride_id": rideID,
		}
		if err := a.wsPort.SendToPassenger(ctx, req.PassengerID, msg); err != nil {
			// Log/return WS failure as domain.ErrWebSocketSend
			return nil, fmt.Errorf("%w: %v", domain.ErrWebSocketSend, err)
		}
	}

	return &domain.RideResponse{
		RideID:                   rideID,
		RideNumber:               rideNumber,
		Status:                   "REQUESTED",
		EstimatedFare:            fare,
		EstimatedDurationMinutes: dur,
		EstimatedDistanceKm:      dist,
	}, nil
}
