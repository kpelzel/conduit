package auth

import (
	"net/http"
	"strings"
)

func BearerTokenFromRequest(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", ErrMissingToken
	}

	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrInvalidToken
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrInvalidToken
	}

	return token, nil
}
