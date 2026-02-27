package handler

import (
	"encoding/json"
	"net/http"
)

// ProvidersConfig holds the configuration that determines which auth providers are enabled.
// These values are derived from environment variables at startup.
type ProvidersConfig struct {
	// GitHubClientID: include "github" when non-empty (GITHUB_CLIENT_ID env var)
	GitHubClientID string
	// DiscordClientID: include "discord" when non-empty (DISCORD_CLIENT_ID env var)
	DiscordClientID string
	// AppleClientID: include "apple" when non-empty (APPLE_CLIENT_ID env var)
	AppleClientID string
	// EnableEmail: include "email" when true (ENABLE_EMAIL_LOGIN=true env var)
	EnableEmail bool
}

// ProvidersHandler handles GET /api/auth/providers
type ProvidersHandler struct {
	cfg ProvidersConfig
}

// NewProvidersHandler creates a ProvidersHandler with the given configuration.
func NewProvidersHandler(cfg ProvidersConfig) *ProvidersHandler {
	return &ProvidersHandler{cfg: cfg}
}

// providersResponse is the JSON response shape for GET /api/auth/providers.
type providersResponse struct {
	Providers []string `json:"providers"`
}

// Providers handles GET /api/auth/providers.
// Google is always included (it is required). Other providers are included
// conditionally based on the ProvidersConfig.
func (h *ProvidersHandler) Providers(w http.ResponseWriter, r *http.Request) {
	// Google is always present and always first.
	providers := []string{"google"}

	if h.cfg.GitHubClientID != "" {
		providers = append(providers, "github")
	}
	if h.cfg.DiscordClientID != "" {
		providers = append(providers, "discord")
	}
	if h.cfg.AppleClientID != "" {
		providers = append(providers, "apple")
	}
	if h.cfg.EnableEmail {
		providers = append(providers, "email")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providersResponse{Providers: providers})
}
