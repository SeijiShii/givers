package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// GoogleUserInfo は Google OAuth から取得するユーザー情報
type GoogleUserInfo struct {
	Sub   string
	Email string
	Name  string
}

// GitHubUserInfo は GitHub OAuth から取得するユーザー情報
type GitHubUserInfo struct {
	ID    int64
	Login string
	Email string
	Name  string
}

// DiscordUserInfo は Discord OAuth から取得するユーザー情報
type DiscordUserInfo struct {
	ID       string
	Username string
	Email    string
}

// AuthService は認証に関するビジネスロジックのインターフェース
type AuthService interface {
	GetOrCreateUserFromGoogle(ctx context.Context, info *GoogleUserInfo) (*model.User, error)
	GetOrCreateUserFromGitHub(ctx context.Context, info *GitHubUserInfo) (*model.User, error)
	GetOrCreateUserFromDiscord(ctx context.Context, info *DiscordUserInfo) (*model.User, error)
}
