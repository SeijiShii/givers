package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// WatchService はウォッチ機能に関するビジネスロジックのインターフェース
type WatchService interface {
	Watch(ctx context.Context, userID, projectID string) error
	Unwatch(ctx context.Context, userID, projectID string) error
	ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error)
}
