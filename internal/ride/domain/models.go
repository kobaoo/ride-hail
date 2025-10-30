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


type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type DriverResponse struct {
	RideID                  string   `json:"ride_id"`
	DriverID                string   `json:"driver_id"`
	Accepted                bool     `json:"accepted"`
	EstimatedArrivalMinutes int      `json:"estimated_arrival_minutes"`
	DriverLocation          Location `json:"driver_location"`
	EstimatedArrival        string   `json:"estimated_arrival"`
	CorrelationID           string   `json:"correlation_id"`
}

type LocationUpdate struct {
	DriverID  string   `json:"driver_id"`
	RideID    string   `json:"ride_id"`
	Location  Location `json:"location"`
	SpeedKmh  float64  `json:"speed_kmh"`
	Heading   float64  `json:"heading_degrees"`
	Timestamp string   `json:"timestamp"`
}

type RideRequest struct {
    PassengerID        string  `json:"passenger_id"`
    PickupLatitude     float64 `json:"pickup_latitude"`
    PickupLongitude    float64 `json:"pickup_longitude"`
    PickupAddress      string  `json:"pickup_address"`
    DestinationLatitude  float64 `json:"destination_latitude"`
    DestinationLongitude float64 `json:"destination_longitude"`
    DestinationAddress string  `json:"destination_address"`
    RideType           string  `json:"ride_type"`
}

type RideResponse struct {
    RideID                 string  `json:"ride_id"`
    RideNumber             string  `json:"ride_number"`
    Status                 string  `json:"status"`
    EstimatedFare          float64 `json:"estimated_fare"`
    EstimatedDurationMinutes int   `json:"estimated_duration_minutes"`
    EstimatedDistanceKm    float64 `json:"estimated_distance_km"`
}