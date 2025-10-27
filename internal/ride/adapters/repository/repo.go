package repository

import (
	"context"
	"ride-hail/internal/ride/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rideRepo struct {
	pool *pgxpool.Pool
}

func NewRideRepository(pool *pgxpool.Pool) domain.RideRepository {
	return &rideRepo{pool: pool}
}

func (r *rideRepo) Insert(ctx context.Context, rd *domain.Ride) error {
	const q = `
		INSERT INTO rides (
			ride_number,
			passenger_id,
			vehicle_type,
			status,
			estimated_fare,
			pickup_coordinate_id,
			destination_coordinate_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id, created_at, updated_at, requested_at`

	return r.pool.QueryRow(ctx, q,
		rd.RideNumber,
		rd.PassengerID,
		rd.VehicleType,
		"REQUESTED",
		rd.EstimatedFare,
		rd.PickupCoordinateID,
		rd.DestinationCoordinateID,
	).Scan(&rd.ID, &rd.CreatedAt, &rd.UpdatedAt, &rd.RequestedAt)
}

func (r *rideRepo) Cancel(ctx context.Context, rideID string, _ domain.RideStatus) error {
	const q = `UPDATE rides SET status='CANCELLED', cancelled_at=NOW() WHERE id=$1`
	ct, err := r.pool.Exec(ctx, q, rideID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
