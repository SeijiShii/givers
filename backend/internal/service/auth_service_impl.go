package service

import (
	"context"
	"fmt"

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
	u, err := s.userRepo.FindByGoogleID(ctx, info.Sub)
	if err == nil {
		return u, nil
	}

	// 存在しない場合は作成
	newUser := &model.User{
		Email:    info.Email,
		GoogleID: info.Sub,
		Name:     info.Name,
	}
	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}
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
