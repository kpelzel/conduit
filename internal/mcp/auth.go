package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	conduitauth "github.com/lanl/conduit/internal/server/httpserver/auth"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func (m *MCPServer) verifyToken(
	ctx context.Context,
	token string,
	_ *http.Request,
) (*mcpauth.TokenInfo, error) {
	p, err := m.validator.ValidateBearerToken(ctx, token, conduitauth.ValidateOptions{
		// Do not pass RequiredScopes here if you want the MCP middleware
		// to generate proper 403 insufficient_scope responses.
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
	}

	if p.Username == "" {
		return nil, fmt.Errorf("%w: missing username", mcpauth.ErrInvalidToken)
	}

	exp := p.ExpiresAt
	if exp.IsZero() {
		// MCP SDK requires a non-zero future expiration.
		// Since this verifier introspects every request, this is just a short
		// per-request compatibility TTL, not token trust beyond introspection.
		exp = time.Now().Add(m.tokenExpirationFallback)
	}

	return &mcpauth.TokenInfo{
		UserID:     p.Username,
		Scopes:     p.Scopes,
		Expiration: exp,
		Extra: map[string]any{
			"subject":   p.Subject,
			"client_id": p.ClientID,
			"audiences": p.Audiences,
			"claims":    p.Claims,
		},
	}, nil
}
