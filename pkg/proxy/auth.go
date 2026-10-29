package proxy

import (
	"net/http"
	"os"
	"strings"
	"sukami/pkg/auth"
)

// SetGatewayKey configures the master authorization key for cloud deployment.
func (h *Handler) SetGatewayKey(key string) {
	h.gatewayKey = strings.TrimSpace(key)
}

// validateGatewayAuth verifies if request satisfies master gateway key or registered tenant key.
func (h *Handler) validateGatewayAuth(r *http.Request) (bool, *auth.APIKey) {
	clientKey := resolveSukamiKey(r)

	// If no gateway master key is configured, check tenant key if provided (local developer mode)
	if h.gatewayKey == "" {
		if clientKey != "" {
			tenant, valid := h.authStore.Validate(clientKey)
			if !valid {
				return false, nil
			}
			return true, tenant
		}
		return true, nil
	}

	// Gateway master key is enforced (cloud security mode)
	if isMasterKeyMatch(r, h.gatewayKey) {
		return true, nil
	}

	// Allow valid tenant keys to pass gateway auth
	if clientKey != "" {
		tenant, valid := h.authStore.Validate(clientKey)
		if valid {
			return true, tenant
		}
	}

	return false, nil
}

// requireMasterAuth checks if caller is authorized for admin management endpoints (/api/keys).
func (h *Handler) requireMasterAuth(r *http.Request) bool {
	if h.gatewayKey == "" {
		return true
	}
	return isMasterKeyMatch(r, h.gatewayKey)
}

func isMasterKeyMatch(r *http.Request, masterKey string) bool {
	if masterKey == "" {
		return false
	}
	for _, header := range []string{"X-Sukami-Master-Key", "X-Sukami-Key"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			clean := strings.TrimPrefix(val, "Bearer ")
			if clean == masterKey {
				return true
			}
		}
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	clean := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	return clean == masterKey
}

// resolveSukamiKey extracts the Sukami tenant key from X-Sukami-Key or Authorization header.
func resolveSukamiKey(r *http.Request) string {
	if sukamiHeader := r.Header.Get("X-Sukami-Key"); sukamiHeader != "" {
		return strings.TrimSpace(strings.TrimPrefix(sukamiHeader, "Bearer "))
	}
	rawAuth := r.Header.Get("Authorization")
	clean := strings.TrimSpace(strings.TrimPrefix(rawAuth, "Bearer "))
	if strings.HasPrefix(clean, "sk-sukami-") {
		return clean
	}
	return ""
}

// resolveUpstreamAuth extracts the upstream provider authorization header.
func resolveUpstreamAuth(r *http.Request) string {
	return (&Handler{}).resolveUpstreamAuth(r)
}

func (h *Handler) resolveUpstreamAuth(r *http.Request) string {
	for _, header := range []string{"X-Upstream-Key", "X-Provider-Key", "X-OpenAI-Api-Key"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			return "Bearer " + strings.TrimPrefix(val, "Bearer ")
		}
	}

	reqAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	cleanAuth := strings.TrimPrefix(reqAuth, "Bearer ")
	if cleanAuth != "" && !strings.HasPrefix(cleanAuth, "sk-sukami-") && cleanAuth != h.gatewayKey {
		return "Bearer " + cleanAuth
	}

	for _, envVar := range []string{"OPENAI_API_KEY", "GROQ_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "GEMINI_API_KEY"} {
		if val := os.Getenv(envVar); val != "" {
			return "Bearer " + val
		}
	}
	return ""
}

// scopedCacheModel returns a tenant-scoped model identifier for cache isolation.
func scopedCacheModel(tenantKey, model string) string {
	if tenantKey == "" {
		return "pub|" + model
	}
	prefix := tenantKey
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return prefix + "|" + model
}
