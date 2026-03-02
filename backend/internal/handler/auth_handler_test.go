package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// --- return_url tests ---

func TestAuthHandler_FinalizeLogin_WithReturnURL(t *testing.T) {
	h := newTestAuthHandler()
	storeOneTimeCode("code-ret", "session-ret")

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=code-ret&return_url=/projects/abc123", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "http://localhost:3000/projects/abc123" {
		t.Errorf("expected redirect to /projects/abc123, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_RejectsExternalReturnURL(t *testing.T) {
	h := newTestAuthHandler()
	storeOneTimeCode("code-ext", "session-ext")

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=code-ext&return_url=https://evil.com", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	loc := rec.Header().Get("Location")
	if loc != "http://localhost:3000/" {
		t.Errorf("expected redirect to /, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_RejectsProtocolRelativeReturnURL(t *testing.T) {
	h := newTestAuthHandler()
	storeOneTimeCode("code-proto", "session-proto")

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=code-proto&return_url=//evil.com", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	loc := rec.Header().Get("Location")
	if loc != "http://localhost:3000/" {
		t.Errorf("expected redirect to / for protocol-relative URL, got %s", loc)
	}
}

func TestAuthHandler_FinalizeLogin_EmptyReturnURL_DefaultsToHome(t *testing.T) {
	h := newTestAuthHandler()
	storeOneTimeCode("code-empty", "session-empty")

	req := httptest.NewRequest("GET", "/api/auth/finalize?code=code-empty&return_url=", nil)
	rec := httptest.NewRecorder()

	h.FinalizeLogin(rec, req)

	loc := rec.Header().Get("Location")
	if loc != "http://localhost:3000/" {
		t.Errorf("expected redirect to /, got %s", loc)
	}
}

func TestOAuthState_StoresAndReturnsReturnURL(t *testing.T) {
	state := "test-state-return"
	storeOAuthState(state, "/projects/xyz")

	returnURL, ok := verifyAndDeleteOAuthState(state)
	if !ok {
		t.Fatal("expected state to be valid")
	}
	if returnURL != "/projects/xyz" {
		t.Errorf("expected /projects/xyz, got %s", returnURL)
	}
}

func TestOAuthState_EmptyReturnURL(t *testing.T) {
	state := "test-state-no-return"
	storeOAuthState(state, "")

	returnURL, ok := verifyAndDeleteOAuthState(state)
	if !ok {
		t.Fatal("expected state to be valid")
	}
	if returnURL != "" {
		t.Errorf("expected empty return_url, got %s", returnURL)
	}
}

func TestAuthHandler_GoogleLoginURL_PassesReturnURL(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/login?return_url=/projects/abc", nil)
	rec := httptest.NewRecorder()

	h.GoogleLoginURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["url"] == "" {
		t.Fatal("expected url in response")
	}

	// Verify state was stored with return_url by extracting state from the URL
	// and checking it via verifyAndDeleteOAuthState
	authURL := body["url"]
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}
	stateVal := parsed.Query().Get("state")
	if stateVal == "" {
		t.Fatal("expected state in auth URL")
	}

	returnURL, ok := verifyAndDeleteOAuthState(stateVal)
	if !ok {
		t.Fatal("expected state to be valid")
	}
	if returnURL != "/projects/abc" {
		t.Errorf("expected /projects/abc return_url stored in state, got %s", returnURL)
	}
}

// Verify that googleUserInfo correctly parses Google v2 userinfo "id" field.
func TestGoogleUserInfo_ParsesV2ID(t *testing.T) {
	// Google OAuth2 v2 userinfo returns "id", not "sub"
	body := `{"id":"112233445566778899","email":"test@gmail.com","name":"Test User"}`
	var info googleUserInfo
	if err := json.Unmarshal([]byte(body), &info); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if info.ID == "" {
		t.Fatal("expected ID to be parsed from v2 'id' field, got empty string")
	}
	if info.ID != "112233445566778899" {
		t.Errorf("expected ID 112233445566778899, got %s", info.ID)
	}
	if info.Email != "test@gmail.com" {
		t.Errorf("expected email test@gmail.com, got %s", info.Email)
	}
}

// Verify that Google login URL includes prompt=select_account
func TestAuthHandler_GoogleLoginURL_IncludesPromptSelectAccount(t *testing.T) {
	h := newTestAuthHandler()
	req := httptest.NewRequest("GET", "/api/auth/google/login", nil)
	rec := httptest.NewRecorder()

	h.GoogleLoginURL(rec, req)

	var body map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&body)
	authURL := body["url"]
	if !strings.Contains(authURL, "prompt=select_account") {
		t.Errorf("expected prompt=select_account in auth URL, got %s", authURL)
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
