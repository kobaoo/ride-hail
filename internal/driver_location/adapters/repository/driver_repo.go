package repository

import (
	"context"
	"fmt"
	"ride-hail/internal/driver_location/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DriverRepository struct {
	db *pgxpool.Pool
}

func NewDriverRepository(db *pgxpool.Pool) *DriverRepository {
	return &DriverRepository{db: db}
}

// StartSession creates a driver_sessions row and returns the session id.
// Uses a transaction to ensure atomicity.
func (r *DriverRepository) StartSession(ctx context.Context, driverID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // safe to call (no-op if committed)
	}()

	var sessionID string
	row := tx.QueryRow(ctx, `
		INSERT INTO driver_sessions (driver_id)
		VALUES ($1)
		RETURNING id
	`, driverID)
	if err := row.Scan(&sessionID); err != nil {
		return "", fmt.Errorf("insert driver_session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return sessionID, nil
}

// UpdateStatus updates drivers.status; returns pgx.ErrNoRows if driver missing.
func (r *DriverRepository) UpdateStatus(ctx context.Context, driverID, status string) error {
	ct, err := r.db.Exec(ctx, `
		UPDATE drivers
		SET status = $2, updated_at = now()
		WHERE id = $1
	`, driverID, status)
	if err != nil {
		return fmt.Errorf("update drivers status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *DriverRepository) EndSession(ctx context.Context, driverID string) (string, domain.SessionSummary, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", domain.SessionSummary{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// --- 1. Find active session ---
	var sessionID string
	row := tx.QueryRow(ctx, `
		SELECT id
		FROM driver_sessions
		WHERE driver_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`, driverID)
	if err := row.Scan(&sessionID); err != nil {
		if err == pgx.ErrNoRows {
			return "", domain.SessionSummary{}, fmt.Errorf("no active session for driver %s", driverID)
		}
		return "", domain.SessionSummary{}, fmt.Errorf("query active session: %w", err)
	}

	// --- 2. Compute total rides + earnings for this session ---
	var ridesCompleted int
	var totalEarnings float64
	err = tx.QueryRow(ctx, `
		SELECT 
			COUNT(*) AS rides_completed,
			COALESCE(SUM(final_fare), 0)
		FROM rides
		WHERE driver_id = $1
		  AND status = 'COMPLETED'
		  AND completed_at >= (
		      SELECT started_at FROM driver_sessions WHERE id = $2
		  )
		  AND (completed_at <= now() OR completed_at IS NOT NULL)
	`, driverID, sessionID).Scan(&ridesCompleted, &totalEarnings)
	if err != nil {
		return "", domain.SessionSummary{}, fmt.Errorf("query rides summary: %w", err)
	}

	// --- 3. Update session record ---
	ct, err := tx.Exec(ctx, `
		UPDATE driver_sessions
		SET 
			ended_at = now(),
			total_rides = $2,
			total_earnings = $3
		WHERE id = $1
	`, sessionID, ridesCompleted, totalEarnings)
	if err != nil {
		return "", domain.SessionSummary{}, fmt.Errorf("update session: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return "", domain.SessionSummary{}, fmt.Errorf("session not found during update")
	}

	// --- 4. Compute duration ---
	var durationHours float64
	err = tx.QueryRow(ctx, `
		SELECT EXTRACT(EPOCH FROM (now() - started_at)) / 3600.0
		FROM driver_sessions
		WHERE id = $1
	`, sessionID).Scan(&durationHours)
	if err != nil {
		return "", domain.SessionSummary{}, fmt.Errorf("compute duration: %w", err)
	}

	// --- 5. Commit ---
	if err := tx.Commit(ctx); err != nil {
		return "", domain.SessionSummary{}, fmt.Errorf("commit tx: %w", err)
	}

	summary := domain.SessionSummary{
		DurationHours:  durationHours,
		RidesCompleted: ridesCompleted,
		Earnings:       totalEarnings,
	}

	return sessionID, summary, nil
}
