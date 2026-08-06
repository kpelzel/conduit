// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"fmt"
	"net/http"
)

// AuthenticateMTLSRequest extracts the username from the client certificate's Common Name
func AuthenticateMTLSRequest(r *http.Request) (*Principal, error) {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
		return nil, fmt.Errorf("no verified client certificate provided")
	}

	cert := r.TLS.VerifiedChains[0][0]

	// Extract username from the certificate's Common Name
	// The CN format should be: "username" or could potentially be "username@realm"
	username := cert.Subject.CommonName
	if username == "" {
		return nil, fmt.Errorf("certificate has no common name")
	}

	principal := &Principal{
		Subject:   cert.Subject.String(),
		Username:  username,
		ClientID:  username,   // Use username as client ID for mTLS
		Scopes:    []string{}, // mTLS doesn't have OAuth scopes
		ExpiresAt: cert.NotAfter,
		Claims: map[string]any{
			"cert_subject": cert.Subject.String(),
			"cert_serial":  cert.SerialNumber.String(),
		},
	}

	return principal, nil
}

// RequireMTLS is a middleware that requires mTLS authentication
func RequireMTLS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := AuthenticateMTLSRequest(r)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Certificate realm="conduit"`)
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := ContextWithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireBearerOrMTLS is a middleware that accepts either Bearer token or mTLS authentication
func RequireBearerOrMTLS(v *Introspector, opts ValidateOptions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prefer a verified client certificate.
		principal, mtlsErr := AuthenticateMTLSRequest(r)
		if mtlsErr == nil {
			ctx := ContextWithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// No usable client certificate; try OAuth.
		principal, oauthErr := AuthenticateRequest(r, v, opts)
		if oauthErr != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", Certificate realm="conduit"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := ContextWithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
