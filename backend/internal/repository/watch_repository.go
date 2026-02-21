package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// WatchRepository はウォッチ永続化のインターフェース
type WatchRepository interface {
	Watch(ctx context.Context, userID, projectID string) error
	Unwatch(ctx context.Context, userID, projectID string) error
	ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error)
}
