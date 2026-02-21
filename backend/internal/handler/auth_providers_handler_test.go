package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestProvidersHandler_GoogleOnly verifies that only google is returned
// when GitHub/Apple are not configured and email login is disabled.
func TestProvidersHandler_GoogleOnly(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "",
		AppleClientID:  "",
		EnableEmail:    false,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d: %v", len(resp.Providers), resp.Providers)
	}
	if resp.Providers[0] != "google" {
		t.Errorf("expected providers[0]=google, got %q", resp.Providers[0])
	}
}

// TestProvidersHandler_GoogleAndGitHub verifies that github is included
// when GITHUB_CLIENT_ID is set.
func TestProvidersHandler_GoogleAndGitHub(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "gh-client-id-xxx",
		AppleClientID:  "",
		EnableEmail:    false,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d: %v", len(resp.Providers), resp.Providers)
	}

	providerSet := make(map[string]bool)
	for _, p := range resp.Providers {
		providerSet[p] = true
	}
	if !providerSet["google"] {
		t.Error("expected google in providers")
	}
	if !providerSet["github"] {
		t.Error("expected github in providers")
	}
}

// TestProvidersHandler_AllProviders verifies all four providers are returned
// when all env vars are set.
func TestProvidersHandler_AllProviders(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "gh-client-id",
		AppleClientID:  "apple-client-id",
		EnableEmail:    true,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) != 4 {
		t.Fatalf("expected 4 providers, got %d: %v", len(resp.Providers), resp.Providers)
	}

	providerSet := make(map[string]bool)
	for _, p := range resp.Providers {
		providerSet[p] = true
	}
	for _, want := range []string{"google", "github", "apple", "email"} {
		if !providerSet[want] {
			t.Errorf("expected %q in providers, got %v", want, resp.Providers)
		}
	}
}

// TestProvidersHandler_GoogleAlwaysFirst verifies google is always the
// first element in the list (canonical ordering).
func TestProvidersHandler_GoogleAlwaysFirst(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "gh-id",
		AppleClientID:  "ap-id",
		EnableEmail:    true,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) == 0 || resp.Providers[0] != "google" {
		t.Errorf("expected google to be first, got %v", resp.Providers)
	}
}

// TestProvidersHandler_ContentTypeJSON verifies the response Content-Type header.
func TestProvidersHandler_ContentTypeJSON(t *testing.T) {
	cfg := ProvidersConfig{}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}
}

// TestProvidersHandler_AppleOnly verifies apple is included when APPLE_CLIENT_ID is set
// but GitHub is not.
func TestProvidersHandler_AppleOnly(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "",
		AppleClientID:  "apple-id",
		EnableEmail:    false,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) != 2 {
		t.Fatalf("expected 2 providers (google + apple), got %d: %v", len(resp.Providers), resp.Providers)
	}

	providerSet := make(map[string]bool)
	for _, p := range resp.Providers {
		providerSet[p] = true
	}
	if !providerSet["google"] {
		t.Error("expected google in providers")
	}
	if !providerSet["apple"] {
		t.Error("expected apple in providers")
	}
	if providerSet["github"] {
		t.Error("did not expect github in providers")
	}
}

// TestProvidersHandler_EmailOnly verifies email is included when ENABLE_EMAIL_LOGIN=true
// but neither GitHub nor Apple client IDs are set.
func TestProvidersHandler_EmailOnly(t *testing.T) {
	cfg := ProvidersConfig{
		GitHubClientID: "",
		AppleClientID:  "",
		EnableEmail:    true,
	}
	h := NewProvidersHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	h.Providers(rec, req)

	var resp struct {
		Providers []string `json:"providers"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Providers) != 2 {
		t.Fatalf("expected 2 providers (google + email), got %d: %v", len(resp.Providers), resp.Providers)
	}

	providerSet := make(map[string]bool)
	for _, p := range resp.Providers {
		providerSet[p] = true
	}
	if !providerSet["email"] {
		t.Error("expected email in providers")
	}
}
