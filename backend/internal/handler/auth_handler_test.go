package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- helpers ---

func newTestAuthHandler() *AuthHandler {
	return NewAuthHandler(nil, AuthConfig{
		GoogleClientID:     "google-client-id",
		GoogleClientSecret: "google-secret",
		GitHubClientID:     "github-client-id",
		GitHubClientSecret: "github-secret",
		GoogleRedirectPath: "/api/auth/google/callback",
		GitHubRedirectPath: "/api/auth/github/callback",
		SessionSecret:      "test-session-secret-must-be-32bytes",
		FrontendURL:        "http://localhost:3000",
	})
}

// --- Tests ---

func TestAuthHandler_GoogleLoginURL_SetsStateCookie(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/login", nil)
	rec := httptest.NewRecorder()

	h.GoogleLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// レスポンス URL に state パラメータが含まれ、"state-placeholder" でないこと
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// oauth_state クッキーが設定されていること
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("expected oauth_state cookie to be set")
	}
	if stateCookie.Value == "" {
		t.Fatal("expected oauth_state cookie to have a non-empty value")
	}
	if stateCookie.Value == "state-placeholder" {
		t.Fatal("state should be random, not placeholder")
	}
	if !stateCookie.HttpOnly {
		t.Error("oauth_state cookie should be HttpOnly")
	}
}

func TestAuthHandler_GitHubLoginURL_SetsStateCookie(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/github/login", nil)
	rec := httptest.NewRecorder()

	h.GitHubLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("expected oauth_state cookie to be set")
	}
	if stateCookie.Value == "" || stateCookie.Value == "state-placeholder" {
		t.Fatal("state should be random, not empty or placeholder")
	}
}

func TestAuthHandler_GoogleCallback_RejectsStateMismatch(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=abc&state=wrong-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct-state"})
	rec := httptest.NewRecorder()

	h.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !containsStr(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func TestAuthHandler_GitHubCallback_RejectsStateMismatch(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=wrong-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct-state"})
	rec := httptest.NewRecorder()

	h.GitHubCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !containsStr(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func TestAuthHandler_GoogleCallback_RejectsMissingStateCookie(t *testing.T) {
	h := newTestAuthHandler()
	// state クッキーなし
	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=abc&state=some-state", nil)
	rec := httptest.NewRecorder()

	h.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !containsStr(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
