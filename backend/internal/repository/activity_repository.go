package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ActivityRepository handles persistence for activity feed events.
type ActivityRepository interface {
	// Insert creates a new activity event.
	Insert(ctx context.Context, a *model.ActivityItem) error
	// ListGlobal returns the most recent activities across all projects.
	ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error)
	// ListByProject returns the most recent activities for a specific project.
	ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error)
}
