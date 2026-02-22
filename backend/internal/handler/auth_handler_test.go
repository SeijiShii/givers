package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/givers/backend/internal/model"
)

// --- mock SessionCreatorDeleter ---

type mockSessionSvc struct {
	createFunc func(ctx context.Context, userID string) (*model.Session, error)
	deleteFunc func(ctx context.Context, token string) error
}

func (m *mockSessionSvc) CreateSession(ctx context.Context, userID string) (*model.Session, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID)
	}
	return &model.Session{Token: "mock-token", UserID: userID}, nil
}

func (m *mockSessionSvc) DeleteSession(ctx context.Context, token string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, token)
	}
	return nil
}

// --- helpers ---

func newTestAuthHandler() *AuthHandler {
	return NewAuthHandler(nil, AuthConfig{
		GoogleClientID:     "google-client-id",
		GoogleClientSecret: "google-secret",
		GitHubClientID:     "github-client-id",
		GitHubClientSecret: "github-secret",
		GoogleRedirectPath: "/api/auth/google/callback",
		GitHubRedirectPath: "/api/auth/github/callback",
		FrontendURL:        "http://localhost:3000",
	}, &mockSessionSvc{})
}

// --- Tests ---

func TestAuthHandler_GoogleLoginURL_ReturnsURLWithState(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/login", nil)
	rec := httptest.NewRecorder()

	h.GoogleLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	url := body["url"]
	if url == "" {
		t.Fatal("expected url in response body")
	}
	if !strings.Contains(url, "state=") {
		t.Error("expected state parameter in auth URL")
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Error("expected redirect_uri parameter in auth URL")
	}

	// No cookies should be set (server-side state)
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			t.Error("should NOT set oauth_state cookie — state is stored server-side")
		}
	}
}

func TestAuthHandler_GitHubLoginURL_ReturnsURLWithState(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/github/login", nil)
	rec := httptest.NewRecorder()

	h.GitHubLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["url"] == "" {
		t.Fatal("expected url in response body")
	}
	if !strings.Contains(body["url"], "state=") {
		t.Error("expected state parameter in auth URL")
	}
}

func TestAuthHandler_GoogleCallback_RejectsUnknownState(t *testing.T) {
	h := newTestAuthHandler()
	// state not stored server-side → should be rejected
	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=abc&state=unknown-state", nil)
	rec := httptest.NewRecorder()

	h.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func TestAuthHandler_GitHubCallback_RejectsUnknownState(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=abc&state=unknown-state", nil)
	rec := httptest.NewRecorder()

	h.GitHubCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func TestAuthHandler_GoogleCallback_RejectsMissingState(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/callback?code=abc", nil)
	rec := httptest.NewRecorder()

	h.GoogleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_state") {
		t.Errorf("expected invalid_state error redirect, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_ValidCode(t *testing.T) {
	h := newTestAuthHandler()

	// Store a one-time code
	storeOneTimeCode("test-code-123", "session-token-abc")

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=test-code-123", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}

	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "http://localhost:3000/") {
		t.Errorf("expected redirect to frontend, got %s", loc)
	}

	// Check session cookie is set
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "givers_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected givers_session cookie to be set")
	}
	if sessionCookie.Value != "session-token-abc" {
		t.Errorf("expected session token session-token-abc, got %s", sessionCookie.Value)
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie should be HttpOnly")
	}
}

func TestAuthHandler_FinalizeLogin_InvalidCode(t *testing.T) {
	h := newTestAuthHandler()

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=invalid-code", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_code") {
		t.Errorf("expected invalid_code error redirect, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_MissingCode(t *testing.T) {
	h := newTestAuthHandler()

	req := httptest.NewRequest("GET", "/api/auth/finalize", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "error=missing_code") {
		t.Errorf("expected missing_code error redirect, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_CodeUsedOnlyOnce(t *testing.T) {
	h := newTestAuthHandler()

	storeOneTimeCode("once-code", "token-xyz")

	// First use should succeed
	req1 := httptest.NewRequest("GET", "/api/auth/finalize?code=once-code", nil)
	rec1 := httptest.NewRecorder()
	h.FinalizeLogin(rec1, req1)
	if rec1.Code != http.StatusFound {
		t.Fatalf("first use: expected 302, got %d", rec1.Code)
	}
	if strings.Contains(rec1.Header().Get("Location"), "error=") {
		t.Fatalf("first use should succeed, got %s", rec1.Header().Get("Location"))
	}

	// Second use should fail
	req2 := httptest.NewRequest("GET", "/api/auth/finalize?code=once-code", nil)
	rec2 := httptest.NewRecorder()
	h.FinalizeLogin(rec2, req2)
	loc := rec2.Header().Get("Location")
	if !strings.Contains(loc, "error=invalid_code") {
		t.Errorf("second use should fail with invalid_code, got %s", loc)
	}
}

func TestAuthHandler_Logout_DeletesSessionAndClearsCookie(t *testing.T) {
	var deletedToken string
	svc := &mockSessionSvc{
		deleteFunc: func(_ context.Context, token string) error {
			deletedToken = token
			return nil
		},
	}
	h := NewAuthHandler(nil, AuthConfig{
		FrontendURL: "http://localhost:3000",
	}, svc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "givers_session", Value: "session-token-123"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if deletedToken != "session-token-123" {
		t.Errorf("expected session deletion for token session-token-123, got %q", deletedToken)
	}
	// Check cookie is cleared
	for _, c := range rec.Result().Cookies() {
		if c.Name == "givers_session" && c.MaxAge != -1 {
			t.Errorf("expected session cookie to be cleared (MaxAge=-1), got MaxAge=%d", c.MaxAge)
		}
	}
}

func TestAuthHandler_Logout_DeleteError_StillClearsCookie(t *testing.T) {
	svc := &mockSessionSvc{
		deleteFunc: func(_ context.Context, _ string) error {
			return errors.New("db error")
		},
	}
	h := NewAuthHandler(nil, AuthConfig{
		FrontendURL: "http://localhost:3000",
	}, svc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "givers_session", Value: "tok"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	// Should still succeed — cookie cleared even if DB delete fails
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
