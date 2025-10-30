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
	"ride-hail/internal/driver_location/app"
	"ride-hail/internal/driver_location/domain"
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
		writeJSONError(ctx, w, http.StatusNotFound, "endpoint not found")
		return
	}

	driverID := parts[0]
	action := parts[1]

	switch {
	case r.Method == http.MethodPost && action == "online":
		h.handleGoOnline(ctx, w, r, driverID)
	case r.Method == http.MethodPost && action == "offline":
		h.handleGoOffline(ctx, w, r, driverID)
	default:
		writeJSONError(ctx, w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// -------------------- DRIVER GO ONLINE --------------------

func (h *Handler) handleGoOnline(ctx context.Context, w http.ResponseWriter, r *http.Request, driverID string) {
	ctx = contextx.WithRequestID(ctx, contextx.GetRequestID(ctx))
	start := time.Now()

	// --- Auth ---
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(ctx, w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.VerifyDriverJWT(token)
	if err != nil {
		writeJSONError(ctx, w, http.StatusUnauthorized, "invalid token")
		return
	}
	if claims.DriverID != driverID {
		writeJSONError(ctx, w, http.StatusForbidden, "forbidden: token does not match driver ID")
		return
	}

	// --- Parse body ---
	var req goOnlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error(ctx, h.logger, "invalid_body", "Unable to decode request body", err)
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// --- Core use case ---
	sessionID, err := h.appService.GoOnline(ctx, driverID, req.Latitude, req.Longitude)
	if err != nil {
		h.handleAppError(ctx, w, err, driverID)
		return
	}

	resp := goOnlineResponse{
		Status:    "AVAILABLE",
		SessionID: sessionID,
		Message:   "You are now online and ready to accept rides",
	}
	writeJSONInfo(ctx, w, http.StatusOK, resp)

	log.Info(ctx, h.logger, "driver_online",
		fmt.Sprintf("driver=%s duration_ms=%d", driverID, time.Since(start).Milliseconds()))
}

// -------------------- DRIVER GO OFFLINE --------------------

func (h *Handler) handleGoOffline(ctx context.Context, w http.ResponseWriter, r *http.Request, driverID string) {
	ctx = contextx.WithRequestID(ctx, contextx.GetRequestID(ctx))
	start := time.Now()

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(ctx, w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.VerifyDriverJWT(token)
	if err != nil {
		writeJSONError(ctx, w, http.StatusUnauthorized, "invalid token")
		return
	}
	if claims.DriverID != driverID {
		writeJSONError(ctx, w, http.StatusForbidden, "forbidden: token does not match driver ID")
		return
	}

	sessionID, summary, err := h.appService.GoOffline(ctx, driverID)
	if err != nil {
		h.handleAppError(ctx, w, err, driverID)
		return
	}

	resp := map[string]any{
		"status":          "OFFLINE",
		"session_id":      sessionID,
		"session_summary": summary,
		"message":         "You are now offline",
	}
	writeJSONInfo(ctx, w, http.StatusOK, resp)

	log.Info(ctx, h.logger, "driver_offline",
		fmt.Sprintf("driver=%s duration_ms=%d", driverID, time.Since(start).Milliseconds()))
}

// -------------------- ERROR HANDLING --------------------

func (h *Handler) handleAppError(ctx context.Context, w http.ResponseWriter, err error, driverID string) {
	switch {
	case errors.Is(err, domain.ErrInvalidCoordinates):
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid coordinates")
	case errors.Is(err, domain.ErrInvalidDriverID):
		writeJSONError(ctx, w, http.StatusBadRequest, "invalid driver ID")
	case errors.Is(err, domain.ErrPublishFailed):
		log.Error(ctx, h.logger, "publish_fail driver", driverID, err)
		writeJSONError(ctx, w, http.StatusInternalServerError, "status publish failed")
	case errors.Is(err, domain.ErrWebSocketSend):
		log.Warn(ctx, h.logger, "ws_send_fail driver", driverID, err)
		writeJSONError(ctx, w, http.StatusAccepted, "status updated but ws notification failed")
	default:
		log.Error(ctx, h.logger, "internal_error driver", driverID, err)
		writeJSONError(ctx, w, http.StatusInternalServerError, "internal server error")
	}
}

// -------------------- RESPONSE HELPERS --------------------

func writeJSONError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]any{
		"error":      message,
		"code":       status,
		"request_id": contextx.GetRequestID(ctx),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeJSONInfo(ctx context.Context, w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
