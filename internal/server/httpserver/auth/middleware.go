// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrInsufficientScope = errors.New("insufficient scope")
)

func AuthenticateRequest(r *http.Request, v *Introspector, opts ValidateOptions) (*Principal, error) {
	token, err := BearerTokenFromRequest(r)
	if err != nil {
		return nil, err
	}

	return v.ValidateBearerToken(r.Context(), token, opts)
}

func RequireBearer(v *Introspector, opts ValidateOptions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := AuthenticateRequest(r, v, opts)
		if err != nil {
			writeAuthError(w, err, opts.RequiredScopes)
			return
		}

		ctx := ContextWithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeAuthError(w http.ResponseWriter, err error, requiredScopes []string) {
	if errors.Is(err, ErrInsufficientScope) {
		challenge := `Bearer error="insufficient_scope"`
		if len(requiredScopes) > 0 {
			challenge += `, scope="` + strings.Join(requiredScopes, " ") + `"`
		}

		w.Header().Set("WWW-Authenticate", challenge)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
