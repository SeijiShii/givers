package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// DB は DB 接続の生存確認を行うインターフェース
type DB interface {
	Ping(ctx context.Context) error
}

// UserRepository はユーザー永続化のインターフェース
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*model.User, error)
	FindByGitHubID(ctx context.Context, githubID string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}
