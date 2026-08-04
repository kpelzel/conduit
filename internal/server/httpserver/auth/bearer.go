// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func BearerTokenFromRequest(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", fmt.Errorf("missing bearer token")
	}

	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", fmt.Errorf("malformed bearer token header")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("malformed token")
	}

	return token, nil
}
