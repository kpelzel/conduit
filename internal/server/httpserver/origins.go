// Copyright 2026. Triad National Security, LLC. All rights reserved.

package httpserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/lanl/conduit/internal/logger"
)

type OriginPolicy struct {
	log     *logger.ConduitLogger
	origins []string
	allowed map[string]struct{}
}

func NewOriginPolicy(log *logger.ConduitLogger, rawOrigins []string) (*OriginPolicy, error) {
	origins := make([]string, 0, len(rawOrigins))
	allowed := make(map[string]struct{})

	for _, raw := range rawOrigins {
		origin, err := normalizeOrigin(raw)
		if err != nil {
			return nil, err
		}

		if _, ok := allowed[origin]; ok {
			continue
		}

		allowed[origin] = struct{}{}
		origins = append(origins, origin)
	}

	return &OriginPolicy{
		log:     log,
		origins: origins,
		allowed: allowed,
	}, nil
}

func (p *OriginPolicy) CORSOrigins() []string {
	return append([]string(nil), p.origins...)
}

func (p *OriginPolicy) CheckWebsocketOrigin(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Origin"))
	if raw == "" {
		return false
	}

	origin, err := normalizeOrigin(raw)
	if err != nil {
		p.log.Warnf("rejecting websocket request with invalid origin %q: %v", raw, err)
		return false
	}

	_, ok := p.allowed[origin]
	if !ok {
		p.log.Warnf("rejecting websocket request from origin %q", origin)
	}

	return ok
}

func normalizeOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("origin is empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid origin %q: %w", raw, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("origin %q must use http or https", raw)
	}

	if u.Host == "" {
		return "", fmt.Errorf("origin %q is missing host", raw)
	}

	if u.User != nil {
		return "", fmt.Errorf("origin %q must not include user info", raw)
	}

	if u.Path != "" && u.Path != "/" {
		return "", fmt.Errorf("origin %q must not include path", raw)
	}

	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("origin %q must not include query or fragment", raw)
	}

	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

func (p *OriginPolicy) CheckRequestOrigin(r *http.Request) bool {
	raw := strings.TrimSpace(r.Header.Get("Origin"))

	// Server-side MCP clients and curl normally omit Origin.
	if raw == "" {
		return true
	}

	origin, err := normalizeOrigin(raw)
	if err != nil {
		p.log.Warnf(
			"rejecting request with invalid origin %q: %v",
			raw,
			err,
		)
		return false
	}

	_, ok := p.allowed[origin]
	if !ok {
		p.log.Warnf(
			"rejecting request from origin %q",
			origin,
		)
	}

	return ok
}
