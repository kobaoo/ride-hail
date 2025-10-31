package app

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func (a *AppService) CancelRide(ctx context.Context, rideID, passengerID, reason string) (map[string]any, error) {
	if strings.TrimSpace(rideID) == "" {
		return nil, domain.ErrInvalidRideID
	}
	if strings.TrimSpace(passengerID) == "" {
		return nil, domain.ErrInvalidPassengerID
	}

	cancelledAt, err := a.rideRepo.CancelRide(ctx, rideID, passengerID, reason)
	if err != nil {
		return nil, fmt.Errorf("cancel ride: %w", err)
	}

	// Publish to RMQ
	if err := a.publisher.PublishStatus(ctx, rideID, "CANCELLED", passengerID); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrPublishFailed, err)
	}

	// Notify via WS
	if a.wsPort != nil {
		msg := map[string]any{
			"type":    "ride_update",
			"status":  "CANCELLED",
			"message": "Your ride has been cancelled",
			"ride_id": rideID,
		}
		_ = a.wsPort.SendToPassenger(ctx, passengerID, msg)
	}

	resp := map[string]any{
		"ride_id":      rideID,
		"status":       "CANCELLED",
		"cancelled_at": cancelledAt.Format(time.RFC3339),
		"message":      "Ride cancelled successfully",
	}
	return resp, nil
}
