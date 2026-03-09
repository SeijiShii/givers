package repository

import (
	"context"
	"fmt"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgDonationAlertRepository struct {
	pool *pgxpool.Pool
}

// NewPgDonationAlertRepository returns a PostgreSQL-backed DonationAlertRepository.
func NewPgDonationAlertRepository(pool *pgxpool.Pool) DonationAlertRepository {
	return &pgDonationAlertRepository{pool: pool}
}

const alertSelectCols = `id, project_id, COALESCE(donation_id, ''), donor_type, donor_id,
	alert_type, severity, status, COALESCE(details::text, '{}'),
	created_at, resolved_at, COALESCE(resolved_by, '')`

func scanAlert(scan func(...any) error) (*model.DonationAlert, error) {
	a := &model.DonationAlert{}
	return a, scan(
		&a.ID, &a.ProjectID, &a.DonationID, &a.DonorType, &a.DonorID,
		&a.AlertType, &a.Severity, &a.Status, &a.Details,
		&a.CreatedAt, &a.ResolvedAt, &a.ResolvedBy,
	)
}

func (r *pgDonationAlertRepository) Create(ctx context.Context, a *model.DonationAlert) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO donation_alerts
		 (project_id, donation_id, donor_type, donor_id, alert_type, severity, details)
		 VALUES ($1, NULLIF($2,''), $3, $4, $5, $6, $7::jsonb)`,
		a.ProjectID, a.DonationID, a.DonorType, a.DonorID,
		a.AlertType, a.Severity, a.Details,
	)
	return err
}

func (r *pgDonationAlertRepository) List(ctx context.Context, status string, limit, offset int) ([]*model.DonationAlert, int, error) {
	where := ""
	args := []any{}
	argIdx := 1

	if status != "" {
		where = fmt.Sprintf(" WHERE status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Count
	var total int
	countQuery := "SELECT COUNT(*) FROM donation_alerts" + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List
	query := fmt.Sprintf(
		"SELECT %s FROM donation_alerts%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		alertSelectCols, where, argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var alerts []*model.DonationAlert
	for rows.Next() {
		a, err := scanAlert(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, a)
	}
	return alerts, total, nil
}

func (r *pgDonationAlertRepository) UpdateStatus(ctx context.Context, id, status, resolvedBy string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE donation_alerts
		 SET status = $2,
		     resolved_at = CASE WHEN $2 = 'resolved' THEN NOW() ELSE resolved_at END,
		     resolved_by = CASE WHEN $2 = 'resolved' THEN NULLIF($3, '') ELSE resolved_by END
		 WHERE id = $1`,
		id, status, resolvedBy,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Detection queries (read from donations table) ---

func (r *pgDonationAlertRepository) CountRecentByDonor(ctx context.Context, projectID, donorType, donorID string, windowMinutes int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM donations
		 WHERE project_id = $1 AND donor_type = $2 AND donor_id = $3
		   AND created_at >= NOW() - make_interval(mins => $4)`,
		projectID, donorType, donorID, windowMinutes,
	).Scan(&count)
	return count, err
}

func (r *pgDonationAlertRepository) DailyTotalByDonor(ctx context.Context, projectID, donorType, donorID string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)::int FROM donations
		 WHERE project_id = $1 AND donor_type = $2 AND donor_id = $3
		   AND created_at >= DATE_TRUNC('day', NOW())
		   AND (refund_status IS NULL OR refund_status != 'completed')`,
		projectID, donorType, donorID,
	).Scan(&total)
	return total, err
}

func (r *pgDonationAlertRepository) RefundCountByDonor(ctx context.Context, projectID, donorType, donorID string, windowDays int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM donations
		 WHERE project_id = $1 AND donor_type = $2 AND donor_id = $3
		   AND refund_status = 'completed'
		   AND created_at >= NOW() - make_interval(days => $4)`,
		projectID, donorType, donorID, windowDays,
	).Scan(&count)
	return count, err
}

func (r *pgDonationAlertRepository) HasRecentAlert(ctx context.Context, projectID, donorType, donorID, alertType string, windowHours int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM donation_alerts
		   WHERE project_id = $1 AND donor_type = $2 AND donor_id = $3
		     AND alert_type = $4 AND created_at >= NOW() - make_interval(hours => $5)
		 )`,
		projectID, donorType, donorID, alertType, windowHours,
	).Scan(&exists)
	return exists, err
}
