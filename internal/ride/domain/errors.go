package domain

import "errors"

var (
	ErrInvalidRideRequest         = errors.New("invalid ride request")
	ErrInvalidRideID              = errors.New("invalid ride id")
	ErrPublishFailed              = errors.New("failed to publish ride status")
	ErrWebSocketSend              = errors.New("failed to send WS message")
	ErrRideNotFoundOrInvalidState = errors.New("ride not found or cannot be cancelled")
	ErrInvalidPassengerID         = errors.New("invalid passenger id")
)
