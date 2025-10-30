package app

import (
	"context"
	"errors"
	"strings"

	"ride-hail/internal/ride/domain"
)

type rideService struct {
	repo      domain.RideRepository
	publisher domain.Publisher
}

func NewRideService(repo domain.RideRepository, pub domain.Publisher) domain.RideService {
	return &rideService{repo: repo, publisher: pub}
}

func (s *rideService) Validate(r *domain.RideRequest) error {
	if r.PickupLatitude < -90 || r.PickupLatitude > 90 {
        return errors.New("invalid pickup latitude")
    }
    if r.PickupLongitude < -180 || r.PickupLongitude > 180 {
        return errors.New("invalid pickup longitude")
    }
    if r.DestinationLatitude < -90 || r.DestinationLatitude > 90 {
        return errors.New("invalid destination latitude")
    }
    if r.DestinationLongitude < -180 || r.DestinationLongitude > 180 {
        return errors.New("invalid destination longitude")
    }

	if strings.TrimSpace(r.PassengerID) == "" {
		return errors.New("invalid passenger id")
	}
    // Validate addresses
    if strings.TrimSpace(r.PickupAddress) == "" {
        return errors.New("pickup address required")
    }
    if strings.TrimSpace(r.DestinationAddress) == "" {
        return errors.New("destination address required")
    }

    // Validate ride type
	r.RideType = strings.ToUpper(r.RideType)
    validTypes := map[string]bool{"ECONOMY": true, "COMFORT": true, "BUSINESS": true}
    if !validTypes[r.RideType] {
        return errors.New("invalid ride type")
    }

    return nil
}

func (s *rideService) CalcFare(r *domain.RideRequest) (float64, float64, int){
	t, _ := tariff[r.RideType]
	distanceKm := haversineKm(r.PickupLatitude, r.PickupLongitude, r.DestinationLatitude, r.DestinationLongitude) * 1.43
	distanceKm = roundToTenths(distanceKm)
	estDurationMin := estimateDurationMin(distanceKm, 25)
	
	raw := (t.Base + t.PerKm * distanceKm + t.PerMin * float64(estDurationMin))
	if raw < t.MinFare {
		raw = t.MinFare
	}
	fare := (raw / 10) * 10 
	return fare, distanceKm, estDurationMin
}

func (s *rideService) CreateRide(ctx context.Context, fare float64, distance float64, in *domain.RideRequest) (*domain.Ride, error) {
	r := &domain.Ride{
		PassengerID: in.PassengerID,
		RideNumber: generateRideNumber(),
		VehicleType: in.RideType,
		Status: "REQUESTED",
		EstimatedFare: &fare,
	}

	if err := s.repo.Insert(ctx, r); err != nil {
		return nil, err
	}
	corrID, _ := ctx.Value("key").(string)

	_ = s.publisher.PublishRideRequest(ctx, map[string]any{
		"ride_id":      r.ID,
		"ride_number":  r.RideNumber,
		"pickup_location": map[string]any{
			"lat": in.PickupLatitude,
			"lng": in.PickupLongitude,
			"address": in.PickupAddress,
		},
		"destination_location": map[string]any{
			"lat": in.DestinationLatitude,
			"lng": in.DestinationLongitude,
			"address": in.DestinationAddress,
		},
		"ride_type":    r.VehicleType,
		"estimated_fare": r.EstimatedFare,
		"max_distance_km": distance,
		"timeout_seconds": 30,
		"correlation_id": corrID,
	}, r.VehicleType, corrID)

	return r, nil
}

// func (s *rideService) GetRide(ctx context.Context, id string) (*domain.Ride, error) {
// 	return s.repo.GetByID(ctx, id)
// }

// func (s *rideService) UpdateRideStatus(ctx context.Context, id string, status domain.RideStatus) (*domain.Ride, error) {
// 	if status == "" {
// 		return nil, errors.New("status required")
// 	}
// 	ride, err := s.repo.UpdateStatus(ctx, id, status)
// 	if err != nil {
// 		return nil, err
// 	}
// 	_ = s.publisher.PublishRideStatus(ctx, map[string]any{
// 		"ride_id":    ride.ID,
// 		"new_status": ride.Status,
// 		"updated_at": time.Now().UTC().Format(time.RFC3339Nano),
// 	}, string(status), ride.ID)

// 	return ride, nil
// }

// func (s *rideService) EstimateETA(ctx context.Context, id string) (int64, error) {
// 	// Простейший мок: читаем поездку и выдаём ETA, если нет — вернём 7 минут.
// 	ride, err := s.repo.GetByID(ctx, id)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if ride.ETASeconds > 0 {
// 		return ride.ETASeconds, nil
// 	}
// 	return 7 * 60, nil
// }
