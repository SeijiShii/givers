package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgPlatformHealthRepository struct {
	pool *pgxpool.Pool
}

// NewPgPlatformHealthRepository returns a PostgreSQL-backed PlatformHealthRepository.
func NewPgPlatformHealthRepository(pool *pgxpool.Pool) PlatformHealthRepository {
	return &pgPlatformHealthRepository{pool: pool}
}

func (r *pgPlatformHealthRepository) Get(ctx context.Context) (*model.PlatformHealth, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT monthly_cost, current_monthly, warning_threshold, critical_threshold, updated_at
		FROM platform_health
		WHERE id = 1
	`)

	h := &model.PlatformHealth{}
	if err := row.Scan(&h.MonthlyCost, &h.CurrentMonthly, &h.WarningThreshold, &h.CriticalThreshold, &h.UpdatedAt); err != nil {
		return nil, err
	}
	return h, nil
}
