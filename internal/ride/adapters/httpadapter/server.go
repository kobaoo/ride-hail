package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"
	"ride-hail/internal/common/log"
	"ride-hail/internal/ride/domain"
)

type Server struct {
	svc    domain.RideService
	log    *slog.Logger
	server *http.Server
	mux    *http.ServeMux
}

func NewServer(svc domain.RideService, log *slog.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{svc: svc, log: log, mux: mux}
	mux.HandleFunc("POST /rides", s.handleCreateRide)
	// mux.HandleFunc("POST /rides/{ride_id}/cancel", s.handleCancelRide)
	return s
}

func (s *Server) Start(ctx context.Context, addr string) error {
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shCtx)
	}()
	return s.server.ListenAndServe()
}

func (s *Server) handleCreateRide(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rideReq := &domain.RideRequest{}
	if err := json.NewDecoder(r.Body).Decode(rideReq); err != nil {
		log.Error(ctx, s.log, "ride_json_decode", "Failed to decode to json", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	if err := s.svc.Validate(rideReq); err != nil {
		log.Error(ctx, s.log, "ride_validation", "Invalid ride request", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	
	fare, distance, duration := s.svc.CalcFare(rideReq)
	ride, err := s.svc.CreateRide(ctx, fare, distance, rideReq)
	if err != nil {
		log.Error(ctx, s.log, "ride_creation", "Failed to create ride", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rideResp := &domain.RideResponse{
		RideID: ride.ID,
		RideNumber: ride.RideNumber,
		Status: ride.Status,
		EstimatedFare: fare,
		EstimatedDurationMinutes: duration,
		EstimatedDistanceKm: distance,
	}
	
	log.Info(ctx, s.log, "ride_created", "Ride successfully created")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rideResp)
}