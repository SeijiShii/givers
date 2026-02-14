package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth_NoCookie_Returns401(t *testing.T) {
	secret := SessionSecretBytes("dev-secret-change-in-production-32bytes")
	mw := RequireAuth(secret)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidToken_Returns401(t *testing.T) {
	secret := SessionSecretBytes("dev-secret-change-in-production-32bytes")
	mw := RequireAuth(secret)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName(), Value: "invalid.token"})
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_ValidToken_CallsNextWithUserID(t *testing.T) {
	secret := SessionSecretBytes("dev-secret-change-in-production-32bytes")
	token := CreateSessionToken("user-123", secret)
	mw := RequireAuth(secret)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName(), Value: token})
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if gotUserID != "user-123" {
		t.Errorf("expected userID=user-123, got %q", gotUserID)
	}
}

func TestDevAuth_SetsDevUserID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Error("userID not in context")
			return
		}
		if userID != DevUserID {
			t.Errorf("expected %q, got %q", DevUserID, userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	DevAuth(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
