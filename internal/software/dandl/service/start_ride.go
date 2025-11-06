package service

import (
	"context"
	"fmt"
	"ride-hail/internal/domain/driver"
	"ride-hail/internal/domain/ride"
	"ride-hail/internal/general/contracts"
	"ride-hail/internal/ports"
	"strings"
	"time"
)

// StartRide transitions the ride to IN_PROGRESS and marks the driver BUSY.
func (service *driverLocationService) StartRide(ctx context.Context, in ports.StartRideInput) (ports.StartRideResult, error) {
	var out ports.StartRideResult
	corrID := generateCorrelationID()

	err := service.uow.WithinTx(ctx, func(ctx context.Context) error {
		// 1. ensure that driver exists
		if _, err := service.drivers.GetByID(ctx, in.DriverID); err != nil {
			return fmt.Errorf("error 1 - driver lookup failed: %w", err)
		}
		// 2. fetch the ride and validate ownership
		r, err := service.rides.GetByID(ctx, in.RideID)
		if err != nil {
			return fmt.Errorf("error 2 - ride lookup failed: %w", err)
		}

		// 3. ensure the caller is the assigned driver.
		fmt.Println("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", r.DriverID, " | || |", in.DriverID)
		if r.DriverID == nil {
			err := service.rides.UpdateDriverID(ctx, in.RideID, in.DriverID)
			if err != nil {
				return fmt.Errorf("error 3.5 - ride %s updating driver id %s", in.RideID, in.DriverID)
			}
			r.DriverID = &in.DriverID
		}

		if *r.DriverID != in.DriverID {
			return fmt.Errorf("error 3 - ride %s is not assignde to driver %s", in.RideID, in.DriverID)
		}

		// 4. transition the ride status to IN_PROGRESS
		if err = r.Start(); err != nil {
			return fmt.Errorf("error 4 - ride start transition failed: %w", err)
		}

		// 5. update ride status -> IN_PROGRESS
		if err := service.rides.UpdateStatus(ctx, in.RideID, ride.StatusInProgress, *r.StartedAt); err != nil {
			return fmt.Errorf("error 5 - ride status update failed: %w", err)
		}

		// 6. update driver status -> BUSY
		if err := service.drivers.UpdateStatus(ctx, in.DriverID, driver.DriverStatusBusy); err != nil {
			return fmt.Errorf("error 6 - driver status update failed: %w", err)
		}

		// prepare output
		out.RideID = in.RideID
		out.Status = driver.DriverStatusBusy.String()
		out.StartedAt = *r.StartedAt
		out.Message = "Ride started successfully"

		return nil
	})
	if err != nil {
		service.logger.Error(ctx, "driver_start_ride_failed", "Failed to start ride", err, map[string]any{
			"driver_id":  in.DriverID,
			"ride_id":    in.RideID,
			"request_id": corrID,
			"error_type": extractErrorNumber(err.Error()), // Добавляем тип ошибки в логи
		})
		return ports.StartRideResult{}, err
	}

	// prepare driver status update message (BUSY)
	statusMsg := contracts.DriverStatusMessage{
		DriverID:  in.DriverID,
		Status:    driver.DriverStatusBusy.String(),
		RideID:    in.RideID,
		Timestamp: time.Now().UTC(),
		Envelope: contracts.Envelope{
			Producer:      "driver-location-service",
			CorrelationID: corrID,
		},
	}

	// 7. publish driver status update (BUSY)
	if err = service.publishDriverStatus(ctx, statusMsg); err != nil {
		service.logger.Error(ctx, "driver_status_publish_failed", "Failed to publish driver status to RabbitMQ", err, map[string]any{
			"driver_id":  in.DriverID,
			"ride_id":    in.RideID,
			"status":     statusMsg.Status,
			"request_id": corrID,
			"error_type": "error 7 - message publishing failed",
		})
		// Note: мы не возвращаем ошибку здесь, так как основная операция уже выполнена
	}

	// log successful start of the ride
	service.logger.Info(ctx, "driver_started_ride", "Driver started ride", map[string]any{
		"driver_id":  in.DriverID,
		"ride_id":    in.RideID,
		"status":     out.Status,
		"started_at": out.StartedAt,
		"request_id": corrID,
	})

	return out, nil
}

// Вспомогательная функция для извлечения номера ошибки из сообщения
func extractErrorNumber(errMsg string) string {
	if strings.Contains(errMsg, "error 1") {
		return "error_1_driver_lookup"
	} else if strings.Contains(errMsg, "error 2") {
		return "error_2_ride_lookup"
	} else if strings.Contains(errMsg, "error 3") {
		return "error_3_driver_assignment"
	} else if strings.Contains(errMsg, "error 4") {
		return "error_4_ride_transition"
	} else if strings.Contains(errMsg, "error 5") {
		return "error_5_ride_status_update"
	} else if strings.Contains(errMsg, "error 6") {
		return "error_6_driver_status_update"
	} else if strings.Contains(errMsg, "error 7") {
		return "error_7_message_publishing"
	}
	return "unknown_error"
}
