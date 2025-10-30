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
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    const q = `
		INSERT INTO rides (
			ride_number,
			passenger_id,
			vehicle_type,
			status,
			estimated_fare
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING id, created_at, updated_at, requested_at`
		
    err = tx.QueryRow(ctx, q, 
		rd.RideNumber,
		rd.PassengerID,
		rd.VehicleType,
		"REQUESTED",
		rd.EstimatedFare,
	).Scan(&rd.ID, &rd.CreatedAt, &rd.UpdatedAt, &rd.RequestedAt)
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}

func (r *rideRepo) Cancel(ctx context.Context, rideID string, _ domain.RideStatus) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    const q = `UPDATE rides SET status='CANCELLED', cancelled_at=NOW() WHERE id=$1`
    ct, err := tx.Exec(ctx, q, rideID)
    if err != nil {
        return err
    }
    if ct.RowsAffected() == 0 {
        return pgx.ErrNoRows
    }

    return tx.Commit(ctx)
}

