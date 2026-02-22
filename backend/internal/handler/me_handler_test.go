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

func TestMeHandler_NoCookie_Returns401(t *testing.T) {
	h := NewMeHandler(&mockUserRepository{}, auth.SessionSecretBytes("test-secret-32-bytes-long-enough"))

	req := httptest.NewRequest("GET", "/api/me", nil)
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMeHandler_InvalidToken_Returns401(t *testing.T) {
	secret := auth.SessionSecretBytes("test-secret-32-bytes-long-enough")
	h := NewMeHandler(&mockUserRepository{}, secret)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: "bad-token"})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMeHandler_UserNotFound_Returns404(t *testing.T) {
	secret := auth.SessionSecretBytes("test-secret-32-bytes-long-enough")
	token := auth.CreateSessionToken("missing-user", secret)

	h := NewMeHandler(&mockUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, errors.New("not found")
		},
	}, secret)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: token})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMeHandler_ValidSession_ReturnsUser(t *testing.T) {
	secret := auth.SessionSecretBytes("test-secret-32-bytes-long-enough")
	token := auth.CreateSessionToken("u1", secret)

	want := &model.User{ID: "u1", Email: "test@example.com", Name: "Test User"}
	h := NewMeHandler(&mockUserRepository{
		findByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			if id == "u1" {
				return want, nil
			}
			return nil, errors.New("not found")
		},
	}, secret)

	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName(), Value: token})
	rec := httptest.NewRecorder()

	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got model.User
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "u1" || got.Email != "test@example.com" {
		t.Errorf("expected user u1, got %+v", got)
	}
}
