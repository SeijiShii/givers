package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// AuthServiceImpl は AuthService の実装
type AuthServiceImpl struct {
	userRepo repository.UserRepository
}

// NewAuthService は AuthServiceImpl を生成する（DI: UserRepository を注入）
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &AuthServiceImpl{userRepo: userRepo}
}

// GetOrCreateUserFromGoogle は Google ユーザー情報からユーザーを取得または作成する
func (s *AuthServiceImpl) GetOrCreateUserFromGoogle(ctx context.Context, info *GoogleUserInfo) (*model.User, error) {
	slog.Debug("get or create google user", "sub", info.Sub, "email", info.Email, "name", info.Name)

	u, err := s.userRepo.FindByGoogleID(ctx, info.Sub)
	if err == nil {
		slog.Debug("google user found by google_id", "user_id", u.ID)
		return u, nil
	}

	// google_id で見つからない場合、同一メールの既存ユーザーを検索して google_id をリンク
	existing, err := s.userRepo.FindByEmail(ctx, info.Email)
	if err == nil {
		slog.Info("google user linked to existing account", "user_id", existing.ID, "email", info.Email)
		existing.GoogleID = info.Sub
		if err := s.userRepo.UpdateProviderID(ctx, existing.ID, "google_id", info.Sub); err != nil {
			slog.Warn("failed to link google_id, returning existing user", "error", err)
		}
		return existing, nil
	}

	// 存在しない場合は新規作成
	newUser := &model.User{
		Email:    info.Email,
		GoogleID: info.Sub,
		Name:     info.Name,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		slog.Error("create google user failed", "error", err)
		return nil, fmt.Errorf("create user: %w", err)
	}
	slog.Info("new user created", "user_id", newUser.ID, "provider", "google")
	return newUser, nil
}

// GetOrCreateUserFromGitHub は GitHub ユーザー情報からユーザーを取得または作成する
func (s *AuthServiceImpl) GetOrCreateUserFromGitHub(ctx context.Context, info *GitHubUserInfo) (*model.User, error) {
	githubID := fmt.Sprintf("%d", info.ID)
	u, err := s.userRepo.FindByGitHubID(ctx, githubID)
	if err == nil {
		return u, nil
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}
	email := info.Email
	if email == "" {
		email = info.Login + "@users.noreply.github.com"
	}

	// 同一メールの既存ユーザーがあれば github_id をリンク
	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		slog.Info("github user linked to existing account", "user_id", existing.ID, "email", email)
		existing.GitHubID = githubID
		if err := s.userRepo.UpdateProviderID(ctx, existing.ID, "github_id", githubID); err != nil {
			slog.Warn("failed to link github_id, returning existing user", "error", err)
		}
		return existing, nil
	}

	newUser := &model.User{
		Email:    email,
		GitHubID: githubID,
		Name:     name,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

// GetOrCreateUserFromDiscord は Discord ユーザー情報からユーザーを取得または作成する
func (s *AuthServiceImpl) GetOrCreateUserFromDiscord(ctx context.Context, info *DiscordUserInfo) (*model.User, error) {
	u, err := s.userRepo.FindByDiscordID(ctx, info.ID)
	if err == nil {
		return u, nil
	}

	name := info.Username
	email := info.Email
	if email == "" {
		email = info.Username + "@discord.invalid"
	}

	// 同一メールの既存ユーザーがあれば discord_id をリンク
	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		existing.DiscordID = info.ID
		if err := s.userRepo.UpdateProviderID(ctx, existing.ID, "discord_id", info.ID); err != nil {
			slog.Warn("failed to link discord_id, returning existing user", "error", err)
		}
		slog.Info("discord user linked to existing account", "user_id", existing.ID, "provider", "discord")
		return existing, nil
	}

	newUser := &model.User{
		Email:     email,
		DiscordID: info.ID,
		Name:      name,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create discord user: %w", err)
	}
	slog.Info("new user created", "user_id", newUser.ID, "provider", "discord")
	return newUser, nil
}
