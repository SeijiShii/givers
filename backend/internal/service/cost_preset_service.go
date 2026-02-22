package service

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// CostPresetService はユーザーコストプリセットのビジネスロジック
type CostPresetService interface {
	List(ctx context.Context, userID string) ([]*model.CostPreset, error)
	Create(ctx context.Context, userID, label, unitType string) (*model.CostPreset, error)
	Update(ctx context.Context, id, userID string, patch model.CostPresetPatch) error
	Delete(ctx context.Context, id, userID string) error
	Reorder(ctx context.Context, userID string, ids []string) error
}

var validUnitTypes = map[string]bool{
	"monthly":     true,
	"daily_x_days": true,
}

// CostPresetServiceImpl は CostPresetService の実装
type CostPresetServiceImpl struct {
	repo repository.CostPresetRepository
}

// NewCostPresetService は CostPresetServiceImpl を生成する
func NewCostPresetService(repo repository.CostPresetRepository) CostPresetService {
	return &CostPresetServiceImpl{repo: repo}
}

// List はユーザーのプリセット一覧を返す。未設定の場合はシステムデフォルトを返す。
func (s *CostPresetServiceImpl) List(ctx context.Context, userID string) ([]*model.CostPreset, error) {
	presets, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(presets) == 0 {
		return model.DefaultCostPresets(), nil
	}
	return presets, nil
}

// Create は新しいプリセットを作成する
func (s *CostPresetServiceImpl) Create(ctx context.Context, userID, label, unitType string) (*model.CostPreset, error) {
	if !validUnitTypes[unitType] {
		return nil, errors.New("invalid unit_type: must be 'monthly' or 'daily_x_days'")
	}
	preset := &model.CostPreset{
		UserID:   userID,
		Label:    label,
		UnitType: unitType,
	}
	if err := s.repo.Create(ctx, preset); err != nil {
		return nil, err
	}
	return preset, nil
}

// Update はプリセットのラベル・単位種別を更新する（所有者のみ）
func (s *CostPresetServiceImpl) Update(ctx context.Context, id, userID string, patch model.CostPresetPatch) error {
	preset, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if preset.UserID != userID {
		return ErrForbidden
	}
	if patch.Label != nil {
		preset.Label = *patch.Label
	}
	if patch.UnitType != nil {
		if !validUnitTypes[*patch.UnitType] {
			return errors.New("invalid unit_type")
		}
		preset.UnitType = *patch.UnitType
	}
	return s.repo.Update(ctx, preset)
}

// Delete はプリセットを削除する（所有者のみ）
func (s *CostPresetServiceImpl) Delete(ctx context.Context, id, userID string) error {
	preset, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if preset.UserID != userID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

// Reorder はプリセットの順序を更新する
func (s *CostPresetServiceImpl) Reorder(ctx context.Context, userID string, ids []string) error {
	return s.repo.Reorder(ctx, userID, ids)
}
