package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/pkg/auth"
)

type mockUserRepository struct {
	findByIDFunc func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}
func (m *mockUserRepository) FindByGoogleID(context.Context, string) (*model.User, error) {
	return nil, errors.New("not found")
}
func (m *mockUserRepository) FindByGitHubID(context.Context, string) (*model.User, error) {
	return nil, errors.New("not found")
}
func (m *mockUserRepository) Create(context.Context, *model.User) error { return nil }
func (m *mockUserRepository) List(context.Context, int, int) ([]*model.User, error) {
	return nil, nil
}
func (m *mockUserRepository) Suspend(context.Context, string, bool) error { return nil }

// mockMeSessionValidator implements auth.SessionValidator for MeHandler tests
type mockMeSessionValidator struct {
	validateFunc func(ctx context.Context, token string) (string, error)
}

func (m *mockMeSessionValidator) ValidateSession(ctx context.Context, token string) (string, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return "", errors.New("invalid")
}

func TestMeHandler_NoCookie_Returns401(t *testing.T) {
	h := NewMeHandler(&mockUserRepository{}, &mockMeSessionValidator{}, nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMeHandler_InvalidToken_Returns401(t *testing.T) {
	sv := &mockMeSessionValidator{
		validateFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("invalid_session")
		},
	}
	h := NewMeHandler(&mockUserRepository{}, sv, nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "bad-token"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMeHandler_UserNotFound_Returns404(t *testing.T) {
	sv := &mockMeSessionValidator{
		validateFunc: func(_ context.Context, token string) (string, error) {
			return "missing-user", nil
		},
	}
	h := NewMeHandler(&mockUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, errors.New("not found")
		},
	}, sv, nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMeHandler_ValidSession_ReturnsUser(t *testing.T) {
	sv := &mockMeSessionValidator{
		validateFunc: func(_ context.Context, token string) (string, error) {
			if token == "valid-token" {
				return "u1", nil
			}
			return "", errors.New("invalid")
		},
	}

	want := &model.User{ID: "u1", Email: "test@example.com", Name: "Test User"}
	h := NewMeHandler(&mockUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			if id == "u1" {
				return want, nil
			}
			return nil, errors.New("not found")
		},
	}, sv, nil)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got meResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "u1" || got.Email != "test@example.com" {
		t.Errorf("expected user u1, got %+v", got)
	}
	if got.Role != "" {
		t.Errorf("expected no role for non-host user, got %q", got.Role)
	}
}

func TestMeHandler_HostEmail_ReturnsHostRole(t *testing.T) {
	sv := &mockMeSessionValidator{
		validateFunc: func(_ context.Context, token string) (string, error) {
			return "u1", nil
		},
	}

	want := &model.User{ID: "u1", Email: "admin@example.com", Name: "Admin"}
	h := NewMeHandler(&mockUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return want, nil
		},
	}, sv, []string{"admin@example.com"})

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got meResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Role != "host" {
		t.Errorf("expected role=host, got %q", got.Role)
	}
}
