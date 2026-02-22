package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseHostEmails_Multiple(t *testing.T) {
	got := ParseHostEmails("admin@example.com,host@givers.co.jp")
	if len(got) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(got))
	}
	if got[0] != "admin@example.com" || got[1] != "host@givers.co.jp" {
		t.Errorf("unexpected emails: %v", got)
	}
}

func TestParseHostEmails_WithSpaces(t *testing.T) {
	got := ParseHostEmails(" admin@example.com , host@givers.co.jp ")
	if len(got) != 2 {
		t.Fatalf("expected 2 emails, got %d", len(got))
	}
	if got[0] != "admin@example.com" || got[1] != "host@givers.co.jp" {
		t.Errorf("unexpected emails: %v", got)
	}
}

func TestParseHostEmails_Empty(t *testing.T) {
	got := ParseHostEmails("")
	if len(got) != 0 {
		t.Errorf("expected 0 emails, got %d", len(got))
	}
}

func TestParseHostEmails_Single(t *testing.T) {
	got := ParseHostEmails("admin@example.com")
	if len(got) != 1 || got[0] != "admin@example.com" {
		t.Errorf("expected [admin@example.com], got %v", got)
	}
}

func TestHostMiddleware_MatchingEmail_SetsHostTrue(t *testing.T) {
	hostEmails := []string{"admin@example.com", "host@givers.co.jp"}
	lookup := func(ctx context.Context, userID string) (string, error) {
		return "admin@example.com", nil
	}

	var gotIsHost bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIsHost = IsHostFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithUserID(req.Context(), "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	HostMiddleware(hostEmails, lookup)(inner).ServeHTTP(rec, req)

	if !gotIsHost {
		t.Error("expected IsHost=true for matching email")
	}
}

func TestHostMiddleware_NonMatchingEmail_SetsHostFalse(t *testing.T) {
	hostEmails := []string{"admin@example.com"}
	lookup := func(ctx context.Context, userID string) (string, error) {
		return "other@example.com", nil
	}

	var gotIsHost bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIsHost = IsHostFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithUserID(req.Context(), "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	HostMiddleware(hostEmails, lookup)(inner).ServeHTTP(rec, req)

	if gotIsHost {
		t.Error("expected IsHost=false for non-matching email")
	}
}

func TestHostMiddleware_NoUserID_SetsHostFalse(t *testing.T) {
	hostEmails := []string{"admin@example.com"}
	lookup := func(ctx context.Context, userID string) (string, error) {
		t.Error("lookup should not be called when no userID")
		return "", nil
	}

	var gotIsHost bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIsHost = IsHostFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	HostMiddleware(hostEmails, lookup)(inner).ServeHTTP(rec, req)

	if gotIsHost {
		t.Error("expected IsHost=false when no userID in context")
	}
}

func TestHostMiddleware_EmptyHostEmails_SetsHostFalse(t *testing.T) {
	lookup := func(ctx context.Context, userID string) (string, error) {
		return "admin@example.com", nil
	}

	var gotIsHost bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIsHost = IsHostFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithUserID(req.Context(), "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	HostMiddleware(nil, lookup)(inner).ServeHTTP(rec, req)

	if gotIsHost {
		t.Error("expected IsHost=false when hostEmails is empty")
	}
}

func TestHostMiddleware_LookupError_SetsHostFalse(t *testing.T) {
	hostEmails := []string{"admin@example.com"}
	lookup := func(ctx context.Context, userID string) (string, error) {
		return "", errors.New("db error")
	}

	var gotIsHost bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIsHost = IsHostFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithUserID(req.Context(), "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	HostMiddleware(hostEmails, lookup)(inner).ServeHTTP(rec, req)

	if gotIsHost {
		t.Error("expected IsHost=false when lookup returns error")
	}
}
