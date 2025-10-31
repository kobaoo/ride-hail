package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"
	"ride-hail/internal/common/auth"
	"ride-hail/internal/common/ws"
	"ride-hail/internal/ride/domain"

	"github.com/gorilla/websocket"
)

type PassengerWSHandler struct {
	logger   *slog.Logger
	hub      *ws.Hub
	upgrader websocket.Upgrader
}

func NewPassengerWSHandler(logger *slog.Logger, hub *ws.Hub) *PassengerWSHandler {
	return &PassengerWSHandler{
		logger: logger,
		hub:    hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (h *PassengerWSHandler) HandlePassengerWS(w http.ResponseWriter, r *http.Request) {
	passengerID := strings.TrimPrefix(r.URL.Path, "/ws/passengers/")
	if passengerID == "" {
		http.Error(w, "missing passenger id", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("ws_upgrade_fail", "error", err)
		return
	}
	defer conn.Close()

	h.logger.Info("ws_connected", "passenger_id", passengerID)

	// --- Step 1: wait for auth within 5 seconds ---
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		h.sendError(conn, "auth timeout or failed to read auth message")
		return
	}

	var authMsg domain.AuthMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil {
		h.sendError(conn, "invalid auth message format")
		return
	}
	if authMsg.Type != "auth" || !strings.HasPrefix(authMsg.Token, "Bearer ") {
		h.sendError(conn, "invalid auth format")
		return
	}

	token := strings.TrimPrefix(authMsg.Token, "Bearer ")
	claims, err := auth.VerifyPassengerJWT(token)
	if err != nil {
		h.sendError(conn, "invalid token")
		return
	}
	if claims.PassengerID != passengerID {
		h.sendError(conn, "token-passenger mismatch")
		return
	}

	// --- Step 2: authenticated connection established ---
	h.logger.Info("ws_auth_success", "passenger_id", passengerID)
	h.hub.Add(passengerID, conn)
	defer h.hub.Remove(passengerID)
	h.sendInfo(conn, "authenticated")

	// --- Step 3: setup ping/pong and read loop ---
	const (
		pingInterval = 30 * time.Second
		pongWait     = 60 * time.Second
	)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			// send periodic ping
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
				h.logger.Warn("ws_ping_fail", "passenger_id", passengerID, "error", err)
				return
			}
		default:
			// wait for client messages (blocks)
			if _, message, err := conn.ReadMessage(); err != nil {
				h.logger.Info("ws_disconnect", "passenger_id", passengerID, "reason", err)
				return
			} else {
				h.logger.Debug("ws_message", "passenger_id", passengerID, "msg", string(message))
			}
		}
	}
}

func (h *PassengerWSHandler) sendError(conn *websocket.Conn, msg string) {
	_ = conn.WriteJSON(domain.ServerMessage{Type: "error", Message: msg})
}

func (h *PassengerWSHandler) sendInfo(conn *websocket.Conn, msg string) {
	_ = conn.WriteJSON(domain.ServerMessage{Type: "info", Message: msg})
}
