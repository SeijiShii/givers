package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// mockSessionRepository is defined in admin_user_service_test.go (same package).

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSessionService_CreateSession_GeneratesTokenAndStores(t *testing.T) {
	var stored *model.Session
	repo := &mockSessionRepository{
		createFunc: func(_ context.Context, s *model.Session) error {
			stored = s
			return nil
		},
	}
	svc := NewSessionService(repo)

	session, err := svc.CreateSession(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Token == "" {
		t.Error("expected non-empty token")
	}
	if len(session.Token) != 64 {
		t.Errorf("expected 64-char hex token, got %d chars", len(session.Token))
	}
	if session.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %q", session.UserID)
	}
	if stored == nil {
		t.Fatal("expected session to be stored in repo")
	}
	if stored.Token != session.Token {
		t.Error("stored token mismatch")
	}
}

func TestSessionService_ValidateSession_ValidToken(t *testing.T) {
	repo := &mockSessionRepository{
		findByTokenFunc: func(_ context.Context, token string) (*model.Session, error) {
			if token == "valid-token" {
				return &model.Session{
					Token:     "valid-token",
					UserID:    "user-1",
					ExpiresAt: time.Now().Add(1 * time.Hour),
				}, nil
			}
			return nil, errors.New("not found")
		},
	}
	svc := NewSessionService(repo)

	userID, err := svc.ValidateSession(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("expected user-1, got %q", userID)
	}
}

func TestSessionService_ValidateSession_ExpiredToken(t *testing.T) {
	var deletedToken string
	repo := &mockSessionRepository{
		findByTokenFunc: func(_ context.Context, token string) (*model.Session, error) {
			return &model.Session{
				Token:     token,
				UserID:    "user-1",
				ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
			}, nil
		},
		deleteByTokenFunc: func(_ context.Context, token string) error {
			deletedToken = token
			return nil
		},
	}
	svc := NewSessionService(repo)

	_, err := svc.ValidateSession(context.Background(), "expired-token")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if deletedToken != "expired-token" {
		t.Errorf("expected expired token to be deleted, got %q", deletedToken)
	}
}

func TestSessionService_ValidateSession_NotFound(t *testing.T) {
	repo := &mockSessionRepository{
		findByTokenFunc: func(_ context.Context, _ string) (*model.Session, error) {
			return nil, errors.New("not found")
		},
	}
	svc := NewSessionService(repo)

	_, err := svc.ValidateSession(context.Background(), "no-such-token")
	if err == nil {
		t.Fatal("expected error for non-existent token")
	}
}

func TestSessionService_DeleteSession(t *testing.T) {
	var deletedToken string
	repo := &mockSessionRepository{
		deleteByTokenFunc: func(_ context.Context, token string) error {
			deletedToken = token
			return nil
		},
	}
	svc := NewSessionService(repo)

	if err := svc.DeleteSession(context.Background(), "tok-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedToken != "tok-1" {
		t.Errorf("expected tok-1, got %q", deletedToken)
	}
}

func TestSessionService_DeleteAllSessions(t *testing.T) {
	var deletedUserID string
	repo := &mockSessionRepository{
		deleteByUserIDFunc: func(_ context.Context, userID string) error {
			deletedUserID = userID
			return nil
		},
	}
	svc := NewSessionService(repo)

	if err := svc.DeleteAllSessions(context.Background(), "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedUserID != "user-1" {
		t.Errorf("expected user-1, got %q", deletedUserID)
	}
}
