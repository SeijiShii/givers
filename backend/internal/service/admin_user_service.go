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
	userRepo repository.UserRepository
}

// NewAdminUserService creates an AdminUserService.
func NewAdminUserService(userRepo repository.UserRepository) AdminUserService {
	return &adminUserService{userRepo: userRepo}
}

func (s *adminUserService) ListUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	return s.userRepo.List(ctx, limit, offset)
}

func (s *adminUserService) SuspendUser(ctx context.Context, id string, suspend bool) error {
	return s.userRepo.Suspend(ctx, id, suspend)
}

func (s *adminUserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}
