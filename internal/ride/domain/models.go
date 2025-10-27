package domain

import "time"

type RideStatus string

const (
	RideStatusRequested RideStatus = "REQUESTED"
	RideStatusCancelled RideStatus = "CANCELLED"
)

type RideType string

const (
	RideTypeEconomy RideType = "ECONOMY"
	RideTypePremium RideType = "PREMIUM"
	RideTypeXL      RideType = "XL"
)

type Ride struct {
	ID                      string     `json:"id"`
	RideNumber              string     `json:"ride_number"`
	PassengerID             string     `json:"passenger_id"`
	DriverID                *string    `json:"driver_id,omitempty"`
	VehicleType             string     `json:"vehicle_type"`
	Status                  string     `json:"status"`
	Priority                int        `json:"priority"`
	RequestedAt             time.Time  `json:"requested_at"`
	MatchedAt               *time.Time `json:"matched_at,omitempty"`
	ArrivedAt               *time.Time `json:"arrived_at,omitempty"`
	StartedAt               *time.Time `json:"started_at,omitempty"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	CancelledAt             *time.Time `json:"cancelled_at,omitempty"`
	CancellationReason      *string    `json:"cancellation_reason,omitempty"`
	EstimatedFare           *float64   `json:"estimated_fare,omitempty"`
	FinalFare               *float64   `json:"final_fare,omitempty"`
	PickupCoordinateID      string     `json:"pickup_coordinate_id"`
	DestinationCoordinateID string     `json:"destination_coordinate_id"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}
