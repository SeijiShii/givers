package service

import (
	"context"
	"errors"
	"log"
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
		log.Printf("[SESSION] CreateSession: FAIL — token generation error: %v", err)
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
		log.Printf("[SESSION] CreateSession: FAIL — DB insert error: %v", err)
		return nil, err
	}
	log.Printf("[SESSION] CreateSession: OK — userID=%s, token=%s..., expiresAt=%v", userID, token[:16], session.ExpiresAt)
	return session, nil
}

// ValidateSession validates a session token and returns the user ID.
// Implements auth.SessionValidator.
func (s *SessionService) ValidateSession(ctx context.Context, token string) (string, error) {
	tokenPrefix := token
	if len(tokenPrefix) > 16 {
		tokenPrefix = tokenPrefix[:16]
	}
	log.Printf("[SESSION] ValidateSession: looking up token=%s... (length=%d)", tokenPrefix, len(token))

	session, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		log.Printf("[SESSION] ValidateSession: FAIL — token not found in DB: %v", err)
		return "", errors.New("invalid_session")
	}
	log.Printf("[SESSION] ValidateSession: found — userID=%s, expiresAt=%v, now=%v", session.UserID, session.ExpiresAt, time.Now())

	if time.Now().After(session.ExpiresAt) {
		log.Printf("[SESSION] ValidateSession: FAIL — session expired")
		_ = s.repo.DeleteByToken(ctx, token)
		return "", errors.New("session_expired")
	}
	log.Printf("[SESSION] ValidateSession: OK — userID=%s", session.UserID)
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
