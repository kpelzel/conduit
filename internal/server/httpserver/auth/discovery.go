// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ProviderMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	IntrospectionEndpoint string `json:"introspection_endpoint"`
	JWKSURI               string `json:"jwks_uri"`

	TokenEndpointAuthMethodsSupported         []string `json:"token_endpoint_auth_methods_supported"`
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported"`
	IntrospectionEndpointAuthSigningAlgs      []string `json:"introspection_endpoint_auth_signing_alg_values_supported"`
}

func DiscoverProviderMetadata(ctx context.Context, httpClient *http.Client, discoveryURL string) (*ProviderMetadata, error) {
	discoveryURL = strings.TrimSpace(discoveryURL)
	if discoveryURL == "" {
		return nil, errors.New("discovery URL is required")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create discovery request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("discovery endpoint [%v] returned %s: %q", discoveryURL, resp.Status, strings.TrimSpace(string(body)))
	}

	var md ProviderMetadata
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&md); err != nil {
		return nil, fmt.Errorf("decode discovery document: %w", err)
	}

	if md.Issuer == "" {
		return nil, errors.New("discovery document missing issuer")
	}

	if md.IntrospectionEndpoint == "" {
		return nil, errors.New("discovery document missing introspection_endpoint")
	}

	// UserInfo is only required if you use userinfo fallback.
	// Do not fail here unless your config requires it.

	return &md, nil
}

func DiscoveryURLFromIssuer(issuer string) string {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	return issuer + "/.well-known/openid-configuration"
}
