package ws

import (
	"context"

	"ride-hail/internal/common/ws"
	"ride-hail/internal/ride/domain"
)

var _ domain.WebSocketPort = (*Talker)(nil)

// Talker wraps the common WS hub to send messages to passengers.
type Talker struct {
	hub *ws.Hub
}

func NewTalker(hub *ws.Hub) *Talker {
	return &Talker{hub: hub}
}

func (t *Talker) SendToPassenger(ctx context.Context, passengerID string, msg any) error {
	return t.hub.Send(passengerID, msg)
}
