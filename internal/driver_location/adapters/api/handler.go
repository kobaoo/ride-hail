package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/contextx"
	"ride-hail/internal/common/log"
	"ride-hail/internal/driver_location/app"
)

type Handler struct {
	appService *app.AppService
	logger     *slog.Logger
}

func NewHandler(appService *app.AppService, lg *slog.Logger) *Handler {
	return &Handler{appService: appService, logger: lg}
}

type goOnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type goOnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/drivers/", h.driversPrefixHandler)
	return mux
}

func (h *Handler) driversPrefixHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithNewRequestID(r.Context())

	p := strings.TrimPrefix(r.URL.Path, "/drivers/")
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	driverID := parts[0]
	action := parts[1]

	switch {
	case r.Method == http.MethodPost && action == "online":
		h.handleGoOnline(ctx, w, r, driverID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGoOnline(ctx context.Context, w http.ResponseWriter, r *http.Request, driverID string) {
	ctx = contextx.WithRequestID(ctx, contextx.GetRequestID(ctx))
	start := time.Now()

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.VerifyDriverJWT(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	if claims.DriverID != driverID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req goOnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, h.logger, "invalid_body", "Unable to decode request body", err)
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	sessionID, err := h.appService.GoOnline(ctx, driverID, req.Latitude, req.Longitude)
	if err != nil {
		log.Error(ctx, h.logger, "go_online_fail", "Failed to execute GoOnline use case", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := goOnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Info(ctx, h.logger, "driver_online", fmt.Sprintf("driver=%s duration_ms=%d", driverID, time.Since(start).Milliseconds()))
}
