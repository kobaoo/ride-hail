package domain

import (
	"errors"
	"math"
)

type RideRequest struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	RideType             string  `json:"ride_type"`
}

func (r *RideRequest) Validate() error {
	if r.PassengerID == "" {
		return ErrInvalidRideRequest
	}
	if r.PickupLatitude < -90 || r.PickupLatitude > 90 ||
		r.DestinationLatitude < -90 || r.DestinationLatitude > 90 {
		return errors.New("invalid latitude")
	}
	if r.PickupLongitude < -180 || r.PickupLongitude > 180 ||
		r.DestinationLongitude < -180 || r.DestinationLongitude > 180 {
		return errors.New("invalid longitude")
	}
	if r.PickupAddress == "" || r.DestinationAddress == "" {
		return errors.New("addresses required")
	}
	return nil
}

func (r *RideRequest) EstimateFare() (fare float64, distKm float64, durMin int) {
	distKm = haversineKm(r.PickupLatitude, r.PickupLongitude,
		r.DestinationLatitude, r.DestinationLongitude)
	durMin = estimateDurationMin(distKm, 25)
	base := 500.0
	perKm := 100.0
	perMin := 50.0
	raw := base + perKm*distKm + perMin*float64(durMin)
	fare = math.Round(raw/10) * 10
	return
}

type RideResponse struct {
	RideID                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
	EstimatedDistanceKm      float64 `json:"estimated_distance_km"`
}

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type ServerMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// --- geometry helpers ---
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func estimateDurationMin(distanceKm, avgSpeedKmh float64) int {
	if avgSpeedKmh <= 1 {
		avgSpeedKmh = 25
	}
	minutes := distanceKm / avgSpeedKmh * 60
	if minutes < 1 {
		minutes = 1
	}
	return int(math.Ceil(minutes))
}
