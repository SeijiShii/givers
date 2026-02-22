package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgCostPresetRepository は CostPresetRepository の PostgreSQL 実装
type PgCostPresetRepository struct {
	pool *pgxpool.Pool
}

// NewPgCostPresetRepository は PgCostPresetRepository を生成する
func NewPgCostPresetRepository(pool *pgxpool.Pool) *PgCostPresetRepository {
	return &PgCostPresetRepository{pool: pool}
}

// ListByUserID はユーザーのコストプリセット一覧を返す
func (r *PgCostPresetRepository) ListByUserID(ctx context.Context, userID string) ([]*model.CostPreset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, label, unit_type, sort_order, created_at, updated_at
		 FROM user_cost_presets WHERE user_id = $1 ORDER BY sort_order, created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []*model.CostPreset
	for rows.Next() {
		var p model.CostPreset
		if err := rows.Scan(&p.ID, &p.UserID, &p.Label, &p.UnitType, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		presets = append(presets, &p)
	}
	return presets, rows.Err()
}

// GetByID は ID でコストプリセットを取得する
func (r *PgCostPresetRepository) GetByID(ctx context.Context, id string) (*model.CostPreset, error) {
	var p model.CostPreset
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, label, unit_type, sort_order, created_at, updated_at
		 FROM user_cost_presets WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.UserID, &p.Label, &p.UnitType, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Create はコストプリセットを作成する
func (r *PgCostPresetRepository) Create(ctx context.Context, preset *model.CostPreset) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO user_cost_presets (user_id, label, unit_type, sort_order)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		preset.UserID, preset.Label, preset.UnitType, preset.SortOrder,
	).Scan(&preset.ID, &preset.CreatedAt, &preset.UpdatedAt)
}

// Update はコストプリセットの label と unit_type を更新する
func (r *PgCostPresetRepository) Update(ctx context.Context, preset *model.CostPreset) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_cost_presets SET label=$1, unit_type=$2, updated_at=NOW() WHERE id=$3`,
		preset.Label, preset.UnitType, preset.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete はコストプリセットを削除する
func (r *PgCostPresetRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM user_cost_presets WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Reorder は ids の順序で sort_order を更新する（ユーザー所有のものだけ）
func (r *PgCostPresetRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for i, id := range ids {
		if _, err := tx.Exec(ctx,
			`UPDATE user_cost_presets SET sort_order=$1, updated_at=NOW() WHERE id=$2 AND user_id=$3`,
			i, id, userID,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
