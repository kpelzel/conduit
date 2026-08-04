// Copyright 2026. Triad National Security, LLC. All rights reserved.

package auth

import (
	"encoding/json"
	"strconv"
	"strings"
)

func stringClaim(claims map[string]any, name string) string {
	v, ok := claims[name]
	if !ok || v == nil {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func firstStringClaim(claims map[string]any, names []string) string {
	for _, name := range names {
		if s := stringClaim(claims, name); s != "" {
			return s
		}
	}

	return ""
}

func int64Claim(claims map[string]any, name string) (int64, bool) {
	v, ok := claims[name]
	if !ok || v == nil {
		return 0, false
	}

	switch x := v.(type) {
	case json.Number:
		i, err := x.Int64()
		return i, err == nil
	case float64:
		return int64(x), true
	case int64:
		return x, true
	case int:
		return int64(x), true
	case string:
		i, err := strconv.ParseInt(x, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func scopesFromClaims(claims map[string]any) []string {
	scopes := scopesFromClaim(claims["scope"])
	if len(scopes) == 0 {
		scopes = stringsFromClaim(claims["scp"])
	}

	return uniqueStrings(scopes)
}

func scopesFromClaim(v any) []string {
	switch x := v.(type) {
	case string:
		return strings.Fields(x)
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func stringsFromClaim(v any) []string {
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		return []string{strings.TrimSpace(x)}
	case []string:
		return uniqueStrings(x)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return uniqueStrings(out)
	default:
		return nil
	}
}

func missingScopes(have []string, required []string) []string {
	missing := make([]string, 0)

	for _, want := range required {
		want = strings.TrimSpace(want)
		if want == "" {
			continue
		}

		if !contains(have, want) {
			missing = append(missing, want)
		}
	}

	return missing
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		out = append(out, v)
	}

	return out
}
