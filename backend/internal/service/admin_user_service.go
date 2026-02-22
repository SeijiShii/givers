package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// AdminUserService provides host-only user management operations.
type AdminUserService interface {
	ListUsers(ctx context.Context, limit, offset int) ([]*model.User, error)
	SuspendUser(ctx context.Context, id string, suspend bool) error
	GetUser(ctx context.Context, id string) (*model.User, error)
}

type adminUserService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository // optional: nil disables session cleanup
}

// NewAdminUserService creates an AdminUserService.
func NewAdminUserService(userRepo repository.UserRepository) AdminUserService {
	return &adminUserService{userRepo: userRepo}
}

// NewAdminUserServiceWithSessions creates an AdminUserService that clears sessions on suspend.
func NewAdminUserServiceWithSessions(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) AdminUserService {
	return &adminUserService{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (s *adminUserService) ListUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return s.userRepo.List(ctx, limit, offset)
}

func (s *adminUserService) SuspendUser(ctx context.Context, id string, suspend bool) error {
	if err := s.userRepo.Suspend(ctx, id, suspend); err != nil {
		return err
	}
	// 停止時はセッション全削除（強制ログアウト）
	if suspend && s.sessionRepo != nil {
		_ = s.sessionRepo.DeleteByUserID(ctx, id)
	}
	return nil
}

func (s *adminUserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}
