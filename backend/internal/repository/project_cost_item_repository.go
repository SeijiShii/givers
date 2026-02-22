package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectCostItemRepository はプロジェクトコスト項目の永続化インターフェース
type ProjectCostItemRepository interface {
	ListByProjectID(ctx context.Context, projectID string) ([]*model.ProjectCostItem, error)
	// ReplaceAll は既存アイテムを全削除して items を挿入し、projects.monthly_target を更新する
	ReplaceAll(ctx context.Context, projectID string, items []*model.ProjectCostItem) error
}
