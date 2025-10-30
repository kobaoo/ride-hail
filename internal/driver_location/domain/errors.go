package domain

import "errors"

var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidDriverID    = errors.New("invalid driver ID")
	ErrPublishFailed      = errors.New("failed to publish driver status")
	ErrWebSocketSend      = errors.New("failed to send WS message")
	ErrAlreadyOnline      = errors.New("driver already online")
	ErrAlreadyOffline     = errors.New("driver already offline")
)
