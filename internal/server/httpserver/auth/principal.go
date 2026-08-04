// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"context"
	"net/http"
	"time"
)

type Principal struct {
	Subject   string
	Username  string // moniker / uid
	ClientID  string
	Scopes    []string
	Audiences []string
	ExpiresAt time.Time

	// Raw introspection/userinfo claims
	Claims map[string]any
}

func (p *Principal) HasScope(scope string) bool {
	return contains(p.Scopes, scope)
}

type principalContextKey struct{}

func ContextWithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(*Principal)
	return p, ok && p != nil
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	p, ok := PrincipalFromContext(ctx)
	if !ok || p.Username == "" {
		return "", false
	}
	return p.Username, true
}

func UsernameFromRequest(r *http.Request) (string, bool) {
	return UsernameFromContext(r.Context())
}
