package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ActivityService provides business logic for the activity feed.
type ActivityService interface {
	Record(ctx context.Context, a *model.ActivityItem) error
	ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error)
	ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error)
}

type activityService struct {
	repo repository.ActivityRepository
}

// NewActivityService creates an ActivityService.
func NewActivityService(repo repository.ActivityRepository) ActivityService {
	return &activityService{repo: repo}
}

func (s *activityService) Record(ctx context.Context, a *model.ActivityItem) error {
	return s.repo.Insert(ctx, a)
}

func (s *activityService) ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
	return s.repo.ListGlobal(ctx, limit)
}

func (s *activityService) ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
	return s.repo.ListByProject(ctx, projectID, limit)
}
