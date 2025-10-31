package domain

import "errors"

var (
	ErrInvalidRideRequest = errors.New("invalid ride request")
	ErrInvalidRideID      = errors.New("invalid ride id")
	ErrPublishFailed      = errors.New("failed to publish ride status")
	ErrWebSocketSend      = errors.New("failed to send WS message")
)
