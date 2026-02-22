package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// SessionRepository handles persistence for user sessions.
type SessionRepository interface {
	Create(ctx context.Context, s *model.Session) error
	FindByToken(ctx context.Context, token string) (*model.Session, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID string) error
}
