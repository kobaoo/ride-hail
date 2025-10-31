package api

import (
	"context"
	"encoding/json"
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

func NewHandler(app *app.AppService, lg *slog.Logger) *Handler {
	return &Handler{appService: app, logger: lg}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rides", h.handleCreateRide)
	return mux
}

// POST /rides
func (h *Handler) handleCreateRide(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithNewRequestID(r.Context())

	if r.Method != http.MethodPost {
		writeJSONError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

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

	rideResp, err := h.appService.CreateRide(ctx, req)
	if err != nil {
		writeJSONError(ctx, w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSONInfo(ctx, w, http.StatusCreated, rideResp)
	log.Info(ctx, h.logger, "ride_created", "ride_id="+rideResp.RideID)
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
