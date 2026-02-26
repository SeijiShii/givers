package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/pkg/auth"
)

// SessionService manages DB-backed user sessions.
// Implements auth.SessionValidator.
type SessionService struct {
	repo repository.SessionRepository
}

// NewSessionService creates a SessionService.
func NewSessionService(repo repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

// CreateSession generates a new opaque token, stores it in DB, and returns the session.
func (s *SessionService) CreateSession(ctx context.Context, userID string) (*model.Session, error) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		slog.Error("session create: token generation failed", "error", err)
		return nil, err
	}
	now := time.Now()
	session := &model.Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(auth.SessionDuration),
	}
	if err := s.repo.Create(ctx, session); err != nil {
		slog.Error("session create: db insert failed", "error", err, "user_id", userID)
		return nil, err
	}
	slog.Debug("session created", "user_id", userID, "token_prefix", token[:16], "expires_at", session.ExpiresAt)
	return session, nil
}

// ValidateSession validates a session token and returns the user ID.
// Implements auth.SessionValidator.
func (s *SessionService) ValidateSession(ctx context.Context, token string) (string, error) {
	tokenPrefix := token
	if len(tokenPrefix) > 16 {
		tokenPrefix = tokenPrefix[:16]
	}
	slog.Debug("session validate: lookup", "token_prefix", tokenPrefix, "token_length", len(token))

	session, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		slog.Debug("session validate: token not found", "error", err)
		return "", errors.New("invalid_session")
	}
	slog.Debug("session validate: found", "user_id", session.UserID, "expires_at", session.ExpiresAt)

	if time.Now().After(session.ExpiresAt) {
		slog.Debug("session validate: expired", "user_id", session.UserID, "expired_at", session.ExpiresAt)
		_ = s.repo.DeleteByToken(ctx, token)
		return "", errors.New("session_expired")
	}
	slog.Debug("session validate: ok", "user_id", session.UserID)
	return session.UserID, nil
}

// DeleteSession removes a session (logout).
func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	return s.repo.DeleteByToken(ctx, token)
}

// DeleteAllSessions removes all sessions for a user (forced logout).
func (s *SessionService) DeleteAllSessions(ctx context.Context, userID string) error {
	return s.repo.DeleteByUserID(ctx, userID)
}
