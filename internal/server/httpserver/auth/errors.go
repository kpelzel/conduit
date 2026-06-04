package auth

import "errors"

var (
	ErrMissingToken      = errors.New("missing bearer token")
	ErrInvalidToken      = errors.New("invalid token")
	ErrInactiveToken     = errors.New("inactive token")
	ErrMissingUsername   = errors.New("missing username claim")
	ErrInsufficientScope = errors.New("insufficient scope")
	ErrInvalidAudience   = errors.New("invalid audience")
)
