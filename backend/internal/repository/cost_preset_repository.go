package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// CostPresetRepository はユーザーコストプリセットの永続化インターフェース
type CostPresetRepository interface {
	ListByUserID(ctx context.Context, userID string) ([]*model.CostPreset, error)
	GetByID(ctx context.Context, id string) (*model.CostPreset, error)
	Create(ctx context.Context, preset *model.CostPreset) error
	Update(ctx context.Context, preset *model.CostPreset) error
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, userID string, ids []string) error
}
