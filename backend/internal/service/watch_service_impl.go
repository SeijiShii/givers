package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// WatchServiceImpl は WatchService の実装
type WatchServiceImpl struct {
	watchRepo repository.WatchRepository
}

// NewWatchService は WatchServiceImpl を生成する（DI: WatchRepository を注入）
func NewWatchService(watchRepo repository.WatchRepository) WatchService {
	return &WatchServiceImpl{watchRepo: watchRepo}
}

// Watch はプロジェクトをウォッチする（冪等）
func (s *WatchServiceImpl) Watch(ctx context.Context, userID, projectID string) error {
	return s.watchRepo.Watch(ctx, userID, projectID)
}

// Unwatch はプロジェクトのウォッチを解除する（冪等）
func (s *WatchServiceImpl) Unwatch(ctx context.Context, userID, projectID string) error {
	return s.watchRepo.Unwatch(ctx, userID, projectID)
}

// ListWatchedProjects はユーザーがウォッチしているプロジェクト一覧を返す
func (s *WatchServiceImpl) ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error) {
	return s.watchRepo.ListWatchedProjects(ctx, userID)
}
