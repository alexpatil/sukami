package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (gw *Gateway) isMasterAuth(r *http.Request) bool {
	if gw.masterKey == "" {
		return true
	}
	for _, header := range []string{"X-Sukami-Master-Key", "X-Sukami-Key"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			if strings.TrimPrefix(val, "Bearer ") == gw.masterKey {
				return true
			}
		}
	}
	rawAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	return strings.TrimPrefix(rawAuth, "Bearer ") == gw.masterKey
}

// CreateKeyRequest represents incoming payload for new tenant key.
type CreateKeyRequest struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

// HandleListKeys returns all registered tenant API keys.
func (gw *Gateway) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	if !gw.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	keys := gw.authStore.ListKeys()
	writeJSON(w, http.StatusOK, keys)
}

// HandleCreateKey provisions a new tenant API key.
func (gw *Gateway) HandleCreateKey(w http.ResponseWriter, r *http.Request) {
	if !gw.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	key, err := gw.authStore.CreateKey(req.Name, req.Tier)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

// HandleDeleteKey revokes an API key.
func (gw *Gateway) HandleDeleteKey(w http.ResponseWriter, r *http.Request) {
	if !gw.isMasterAuth(r) {
		writeError(w, http.StatusUnauthorized, "Unauthorized: Master API key required")
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "Missing key query parameter")
		return
	}

	revoked := gw.authStore.RevokeKey(key)
	writeJSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}
