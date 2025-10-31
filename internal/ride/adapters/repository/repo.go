package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ride-hail/internal/ride/domain"
)

type RideRepository struct {
	db *pgxpool.Pool
}

func NewRideRepository(db *pgxpool.Pool) *RideRepository {
	return &RideRepository{db: db}
}

// Create inserts pickup & destination coordinates then the ride row.
// Returns ride ID (uuid).
func (r *RideRepository) Create(ctx context.Context, req domain.RideRequest, fare float64, rideNumber string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert pickup coordinate
	var pickupCoordID string
	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude,
			created_at, updated_at, is_current
		)
		VALUES (gen_random_uuid(), $1, 'passenger', $2, $3, $4, now(), now(), true)
		RETURNING id
	`, req.PassengerID, req.PickupAddress, req.PickupLatitude, req.PickupLongitude).Scan(&pickupCoordID)
	if err != nil {
		return "", fmt.Errorf("insert pickup coordinate: %w", err)
	}

	// Insert destination coordinate
	var destCoordID string
	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates (
			id, entity_id, entity_type, address, latitude, longitude,
			created_at, updated_at, is_current
		)
		VALUES (gen_random_uuid(), $1, 'passenger', $2, $3, $4, now(), now(), true)
		RETURNING id
	`, req.PassengerID, req.DestinationAddress, req.DestinationLatitude, req.DestinationLongitude).Scan(&destCoordID)
	if err != nil {
		return "", fmt.Errorf("insert destination coordinate: %w", err)
	}

	// Insert ride referencing coordinates
	var rideID string
	err = tx.QueryRow(ctx, `
		INSERT INTO rides (
			ride_number,
			passenger_id,
			driver_id,
			vehicle_type,
			status,
			requested_at,
			estimated_fare,
			pickup_coordinate_id,
			destination_coordinate_id,
			created_at, updated_at
		) VALUES (
			$1, $2, NULL, $3, 'REQUESTED', now(), $4, $5, $6, now(), now()
		) RETURNING id
	`, rideNumber, req.PassengerID, req.RideType, fare, pickupCoordID, destCoordID).Scan(&rideID)
	if err != nil {
		return "", fmt.Errorf("insert ride: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return rideID, nil
}

func (r *RideRepository) CancelRide(ctx context.Context, rideID, passengerID, reason string) (time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var cancelledAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE rides
		SET status = 'CANCELLED',
		    cancelled_at = now(),
		    cancellation_reason = $3,
		    updated_at = now()
		WHERE id = $1 AND passenger_id = $2 AND status NOT IN ('COMPLETED', 'CANCELLED')
		RETURNING cancelled_at
	`, rideID, passengerID, reason).Scan(&cancelledAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, domain.ErrRideNotFoundOrInvalidState
		}
		return time.Time{}, fmt.Errorf("update cancel: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, fmt.Errorf("commit: %w", err)
	}
	return cancelledAt, nil
}
