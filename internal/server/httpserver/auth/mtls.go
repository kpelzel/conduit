// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// AuthenticateMTLSRequest extracts the username from the client certificate's Common Name
func AuthenticateMTLSRequest(r *http.Request) (*Principal, error) {
	if r.TLS == nil {
		return nil, fmt.Errorf("no TLS connection")
	}

	if len(r.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate provided")
	}

	cert := r.TLS.PeerCertificates[0]

	for _, pc := range r.TLS.PeerCertificates {
		logrus.Infof("peer cert: %v", pc.Subject.CommonName)
	}

	// Extract username from the certificate's Common Name
	// The CN format should be: "username" or could potentially be "username@realm"
	username := cert.Subject.CommonName
	if username == "" {
		return nil, fmt.Errorf("certificate has no common name")
	}

	// If the CN contains a realm (username@REALM), extract just the username
	if idx := strings.Index(username, "@"); idx != -1 {
		username = username[:idx]
	}

	// Verify the certificate is valid
	if err := verifyCertificate(cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
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

// verifyCertificate performs basic validation on the client certificate
func verifyCertificate(cert *x509.Certificate) error {
	// The TLS handshake has already verified the certificate chain,
	// so we just need to check basic properties

	if cert.Subject.CommonName == "" {
		return fmt.Errorf("certificate missing common name")
	}

	return nil
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
		var principal *Principal
		var authErr error

		// Try Bearer token first
		token, err := BearerTokenFromRequest(r)
		if err == nil && token != "" {
			principal, authErr = v.ValidateBearerToken(r.Context(), token, opts)
		} else {
			// If no Bearer token, try mTLS
			principal, authErr = AuthenticateMTLSRequest(r)
		}

		if authErr != nil {
			// Set both authentication challenges
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", Certificate realm="conduit"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := ContextWithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
