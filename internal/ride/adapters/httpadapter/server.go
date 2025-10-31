package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/contextx"
	"ride-hail/internal/common/log"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/domain"
)

type Handler struct {
	appService *app.AppService
	logger     *slog.Logger
}

func NewHandler(appService *app.AppService, lg *slog.Logger) *Handler {
	return &Handler{appService: appService, logger: lg}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rides", h.handleCreateRide)
	mux.HandleFunc("/rides/", h.handleRideActions)
	return mux
}

// POST /rides — existing create ride
func (h *Handler) handleCreateRide(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx := contextx.WithNewRequestID(r.Context())
	start := time.Now()

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(ctx, w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.VerifyPassengerJWT(token)
	if err != nil {
		writeJSONError(ctx, w, http.StatusUnauthorized, "invalid token")
		return
	}

	var req domain.RideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.PassengerID = claims.PassengerID

	resp, err := h.appService.CreateRide(ctx, req)
	if err != nil {
		log.Error(ctx, h.logger, "create_ride_fail", "Failed to create ride", err)
		writeJSONError(ctx, w, http.StatusInternalServerError, "failed to create ride")
		return
	}

	writeJSONInfo(ctx, w, http.StatusCreated, resp)
	log.Info(ctx, h.logger, "ride_created",
		fmt.Sprintf("ride=%s duration_ms=%d", resp.RideID, time.Since(start).Milliseconds()))
}

// POST /rides/{ride_id}/cancel
func (h *Handler) handleRideActions(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/rides/") {
		writeJSONError(r.Context(), w, http.StatusNotFound, "not found")
		return
	}
	ctx := contextx.WithNewRequestID(r.Context())

	path := strings.TrimPrefix(r.URL.Path, "/rides/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeJSONError(ctx, w, http.StatusNotFound, "invalid path")
		return
	}

	rideID := parts[0]
	action := parts[1]

	if action == "cancel" && r.Method == http.MethodPost {
		h.handleCancelRide(ctx, w, r, rideID)
		return
	}

	writeJSONError(ctx, w, http.StatusNotFound, "unsupported action")
}

type cancelRideRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) handleCancelRide(ctx context.Context, w http.ResponseWriter, r *http.Request, rideID string) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(ctx, w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.VerifyPassengerJWT(token)
	if err != nil {
		writeJSONError(ctx, w, http.StatusUnauthorized, "invalid token")
		return
	}

	var req cancelRideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.appService.CancelRide(ctx, rideID, claims.PassengerID, req.Reason)
	if err != nil {
		h.handleAppError(ctx, w, err, rideID)
		return
	}

	writeJSONInfo(ctx, w, http.StatusOK, result)
}

// -------------------- ERROR HANDLING --------------------

func (h *Handler) handleAppError(ctx context.Context, w http.ResponseWriter, err error, rideID string) {
	switch {
	case errors.Is(err, domain.ErrInvalidRideID):
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid ride ID")

	case errors.Is(err, domain.ErrInvalidPassengerID):
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid passenger ID")

	case errors.Is(err, domain.ErrRideNotFoundOrInvalidState):
		writeJSONError(ctx, w, http.StatusConflict, "ride not found or cannot be cancelled")

	case errors.Is(err, domain.ErrPublishFailed):
		log.Error(ctx, h.logger, "rmq_publish_fail", fmt.Sprintf("ride=%s", rideID), err)
		writeJSONError(ctx, w, http.StatusInternalServerError, "failed to publish ride status")

	case errors.Is(err, domain.ErrWebSocketSend):
		log.Warn(ctx, h.logger, "ws_notify_fail", fmt.Sprintf("ride=%s", rideID), err)
		writeJSONError(ctx, w, http.StatusAccepted, "cancelled but websocket notification failed")

	default:
		log.Error(ctx, h.logger, "internal_error", fmt.Sprintf("ride=%s", rideID), err)
		writeJSONError(ctx, w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSONError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error":      message,
		"code":       status,
		"request_id": contextx.GetRequestID(ctx),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSONInfo(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
