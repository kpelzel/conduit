package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lanl/conduit/defaults"
)

type Config struct {
	// Prefer DiscoveryURL. If DiscoveryURL is empty and Issuer is set,
	// NewIntrospector can derive Issuer + "/.well-known/openid-configuration".
	DiscoveryURL string
	Issuer       string

	// These can be filled manually or discovered from metadata.
	IntrospectionURL string
	UserInfoURL      string

	ClientID     string
	ClientSecret string

	ExpectedAudience string
	RequiredScopes   []string

	UsernameClaims []string

	UseUserInfoFallback bool

	HTTPClient *http.Client
	Now        func() time.Time
}

type Introspector struct {
	cfg        Config
	httpClient *http.Client
	now        func() time.Time
}

type ValidateOptions struct {
	RequiredScopes   []string
	ExpectedAudience string
}

func NewIntrospector(ctx context.Context, cfg Config) (*Introspector, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	if cfg.IntrospectionURL == "" || cfg.UserInfoURL == "" || cfg.Issuer == "" {
		discoveryURL := cfg.DiscoveryURL
		if discoveryURL == "" && cfg.Issuer != "" {
			discoveryURL = DiscoveryURLFromIssuer(cfg.Issuer)
		}

		if discoveryURL != "" {
			md, err := DiscoverProviderMetadata(ctx, client, discoveryURL)
			if err != nil {
				return nil, err
			}

			if cfg.Issuer != "" && md.Issuer != cfg.Issuer {
				return nil, fmt.Errorf(
					"discovery issuer mismatch: expected %q, got %q",
					cfg.Issuer,
					md.Issuer,
				)
			}

			if cfg.Issuer == "" {
				cfg.Issuer = md.Issuer
			}

			if cfg.IntrospectionURL == "" {
				cfg.IntrospectionURL = md.IntrospectionEndpoint
			}

			if cfg.UserInfoURL == "" {
				cfg.UserInfoURL = md.UserInfoEndpoint
			}
		}
	}

	if cfg.IntrospectionURL == "" {
		return nil, fmt.Errorf("introspection URL is required")
	}

	if cfg.UseUserInfoFallback && cfg.UserInfoURL == "" {
		return nil, fmt.Errorf("userinfo URL is required when userinfo fallback is enabled")
	}

	if len(cfg.UsernameClaims) == 0 {
		cfg.UsernameClaims = defaults.DefaultUsernameClaims
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &Introspector{
		cfg:        cfg,
		httpClient: client,
		now:        now,
	}, nil
}

func (v *Introspector) ValidateBearerToken(ctx context.Context, rawToken string, opts ValidateOptions) (*Principal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, ErrMissingToken
	}

	claims, err := v.introspect(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	active, _ := claims["active"].(bool)
	if !active {
		return nil, ErrInactiveToken
	}

	p := principalFromClaims(claims, v.cfg.UsernameClaims)

	if err := v.validatePrincipal(p, opts); err != nil {
		return nil, err
	}

	if p.Username == "" && v.cfg.UseUserInfoFallback {
		if err := v.enrichFromUserInfo(ctx, rawToken, p); err != nil {
			return nil, err
		}
	}

	if p.Username == "" {
		return nil, ErrMissingUsername
	}

	return p, nil
}

func (v *Introspector) introspect(ctx context.Context, rawToken string) (map[string]any, error) {
	form := url.Values{}
	form.Set("token", rawToken)
	form.Set("token_type_hint", "access_token")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		v.cfg.IntrospectionURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("create introspection request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Most IdPs expect client_secret_basic for introspection.
	req.SetBasicAuth(v.cfg.ClientID, v.cfg.ClientSecret)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call introspection endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("introspection client authentication failed: %w", ErrInvalidToken)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("introspection failed: status=%s body=%q", resp.Status, strings.TrimSpace(string(body)))
	}

	var claims map[string]any
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	dec.UseNumber()

	if err := dec.Decode(&claims); err != nil {
		return nil, fmt.Errorf("decode introspection response: %w", err)
	}

	return claims, nil
}

func (v *Introspector) validatePrincipal(p *Principal, opts ValidateOptions) error {
	if !p.ExpiresAt.IsZero() && !p.ExpiresAt.After(v.now()) {
		return ErrInvalidToken
	}

	requiredScopes := append([]string{}, v.cfg.RequiredScopes...)
	requiredScopes = append(requiredScopes, opts.RequiredScopes...)

	if missing := missingScopes(p.Scopes, requiredScopes); len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrInsufficientScope, strings.Join(missing, ","))
	}

	expectedAudience := opts.ExpectedAudience
	if expectedAudience == "" {
		expectedAudience = v.cfg.ExpectedAudience
	}

	if expectedAudience != "" && !contains(p.Audiences, expectedAudience) {
		return ErrInvalidAudience
	}

	return nil
}

// enrichFromUserInfo will retrieve the user info from the idp's userinfo_endpoint using the provided bearer token
func (v *Introspector) enrichFromUserInfo(ctx context.Context, rawToken string, p *Principal) error {
	if v.cfg.UserInfoURL == "" {
		return ErrMissingUsername
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.UserInfoURL, nil)
	if err != nil {
		return fmt.Errorf("create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+rawToken)
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call userinfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ErrMissingUsername
	}

	var claims map[string]any
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	dec.UseNumber()

	if err := dec.Decode(&claims); err != nil {
		return fmt.Errorf("decode userinfo response: %w", err)
	}

	userInfoSub := stringClaim(claims, "sub")
	if p.Subject != "" && userInfoSub != "" && p.Subject != userInfoSub {
		return ErrInvalidToken
	}

	if p.Claims == nil {
		p.Claims = map[string]any{}
	}

	for k, v := range claims {
		if _, exists := p.Claims[k]; !exists {
			p.Claims[k] = v
		}
	}

	if p.Subject == "" {
		p.Subject = userInfoSub
	}

	if p.Username == "" {
		p.Username = firstStringClaim(claims, v.cfg.UsernameClaims)
	}

	return nil
}

func principalFromClaims(claims map[string]any, usernameClaims []string) *Principal {
	p := &Principal{
		Subject:   stringClaim(claims, "sub"),
		Username:  firstStringClaim(claims, usernameClaims),
		ClientID:  stringClaim(claims, "client_id"),
		Scopes:    scopesFromClaims(claims),
		Audiences: stringsFromClaim(claims["aud"]),
		Claims:    claims,
	}

	if exp, ok := int64Claim(claims, "exp"); ok && exp > 0 {
		p.ExpiresAt = time.Unix(exp, 0)
	}

	return p
}

func (v *Introspector) Issuer() string {
	return v.cfg.Issuer
}
