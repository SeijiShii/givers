package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// PlatformHealthService provides access to platform health data.
type PlatformHealthService interface {
	Get(ctx context.Context) (*model.PlatformHealth, error)
}

type platformHealthService struct {
	repo repository.PlatformHealthRepository
}

// NewPlatformHealthService creates a PlatformHealthService.
func NewPlatformHealthService(repo repository.PlatformHealthRepository) PlatformHealthService {
	return &platformHealthService{repo: repo}
}

func (s *platformHealthService) Get(ctx context.Context) (*model.PlatformHealth, error) {
	return s.repo.Get(ctx)
}
