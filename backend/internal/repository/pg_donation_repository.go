package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgDonationRepository struct {
	pool *pgxpool.Pool
}

// NewPgDonationRepository returns a PostgreSQL-backed DonationRepository.
func NewPgDonationRepository(pool *pgxpool.Pool) DonationRepository {
	return &pgDonationRepository{pool: pool}
}

const donationSelectCols = `id, project_id, donor_type, donor_id, amount, currency,
	COALESCE(message, ''), is_recurring, COALESCE(stripe_payment_id, ''),
	COALESCE(stripe_subscription_id, ''), paused, COALESCE(next_billing_message, ''),
	created_at, updated_at`

func scanDonation(scan func(...any) error) (*model.Donation, error) {
	d := &model.Donation{}
	return d, scan(
		&d.ID, &d.ProjectID, &d.DonorType, &d.DonorID,
		&d.Amount, &d.Currency, &d.Message,
		&d.IsRecurring, &d.StripePaymentID, &d.StripeSubscriptionID,
		&d.Paused, &d.NextBillingMessage, &d.CreatedAt, &d.UpdatedAt,
	)
}

func (r *pgDonationRepository) Create(ctx context.Context, d *model.Donation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO donations
		 (project_id, donor_type, donor_id, amount, currency, message, is_recurring,
		  stripe_payment_id, stripe_subscription_id)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), $7, NULLIF($8,''), NULLIF($9,''))`,
		d.ProjectID, d.DonorType, d.DonorID, d.Amount, d.Currency,
		d.Message, d.IsRecurring, d.StripePaymentID, d.StripeSubscriptionID,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrDuplicate
	}
	return err
}

func (r *pgDonationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+donationSelectCols+`
		 FROM donations
		 WHERE donor_type = 'user' AND donor_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.Donation
	for rows.Next() {
		d, err := scanDonation(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *pgDonationRepository) GetByID(ctx context.Context, id string) (*model.Donation, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+donationSelectCols+` FROM donations WHERE id = $1`, id)
	d, err := scanDonation(row.Scan)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *pgDonationRepository) Patch(ctx context.Context, id string, patch model.DonationPatch) error {
	if patch.Amount == nil && patch.Paused == nil && patch.NextBillingMessage == nil {
		return nil
	}

	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if patch.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", argIdx))
		args = append(args, *patch.Amount)
		argIdx++
	}
	if patch.Paused != nil {
		setClauses = append(setClauses, fmt.Sprintf("paused = $%d", argIdx))
		args = append(args, *patch.Paused)
		argIdx++
	}
	if patch.NextBillingMessage != nil {
		setClauses = append(setClauses, fmt.Sprintf("next_billing_message = NULLIF($%d, '')", argIdx))
		args = append(args, *patch.NextBillingMessage)
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE donations SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgDonationRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM donations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgDonationRepository) GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+donationSelectCols+` FROM donations WHERE stripe_subscription_id = $1`, subscriptionID)
	return scanDonation(row.Scan)
}

func (r *pgDonationRepository) DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM donations WHERE stripe_subscription_id = $1`, subscriptionID)
	return err
}

func (r *pgDonationRepository) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE donations SET donor_type = 'user', donor_id = $1, updated_at = NOW()
		 WHERE donor_type = 'token' AND donor_id = $2`,
		userID, token)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *pgDonationRepository) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*model.Donation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+donationSelectCols+`
		 FROM donations
		 WHERE project_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.Donation
	for rows.Next() {
		d, err := scanDonation(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *pgDonationRepository) ListMessagesByProject(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
	// Count total matching messages
	countQuery := `SELECT COUNT(*) FROM donations d
		LEFT JOIN users u ON d.donor_type = 'user' AND d.donor_id = u.id
		WHERE d.project_id = $1 AND d.message IS NOT NULL AND d.message != ''`
	countArgs := []any{projectID}
	argIdx := 2
	if donor != "" {
		countQuery += fmt.Sprintf(` AND COALESCE(u.display_name, 'Anonymous') ILIKE '%%' || $%d || '%%'`, argIdx)
		countArgs = append(countArgs, donor)
		argIdx++
	}
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch messages
	sortDir := "DESC"
	if sort == "asc" {
		sortDir = "ASC"
	}
	query := fmt.Sprintf(`SELECT COALESCE(u.display_name, 'Anonymous'), d.amount, d.message, d.created_at, d.is_recurring
		FROM donations d
		LEFT JOIN users u ON d.donor_type = 'user' AND d.donor_id = u.id
		WHERE d.project_id = $1 AND d.message IS NOT NULL AND d.message != ''`)
	args := []any{projectID}
	argIdx = 2
	if donor != "" {
		query += fmt.Sprintf(` AND COALESCE(u.display_name, 'Anonymous') ILIKE '%%' || $%d || '%%'`, argIdx)
		args = append(args, donor)
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY d.created_at %s LIMIT $%d OFFSET $%d`, sortDir, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*model.DonationMessage
	for rows.Next() {
		m := &model.DonationMessage{}
		if err := rows.Scan(&m.DonorName, &m.Amount, &m.Message, &m.CreatedAt, &m.IsRecurring); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if msgs == nil {
		msgs = []*model.DonationMessage{}
	}
	return &model.DonationMessageResult{Messages: msgs, Total: total}, nil
}

// CurrentMonthSumByProject returns the total donation amount for a project in the current month.
func (r *pgDonationRepository) CurrentMonthSumByProject(ctx context.Context, projectID string) (int, error) {
	var sum int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)::int
		 FROM donations
		 WHERE project_id = $1
		   AND created_at >= DATE_TRUNC('month', NOW())`,
		projectID,
	).Scan(&sum)
	return sum, err
}

func (r *pgDonationRepository) MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month,
		        SUM(amount)::int AS amount
		 FROM donations
		 WHERE project_id = $1
		   AND created_at >= DATE_TRUNC('month', NOW()) - INTERVAL '11 months'
		 GROUP BY DATE_TRUNC('month', created_at)
		 ORDER BY month`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sums []*model.MonthlySum
	for rows.Next() {
		s := &model.MonthlySum{}
		if err := rows.Scan(&s.Month, &s.Amount); err != nil {
			return nil, err
		}
		sums = append(sums, s)
	}
	return sums, rows.Err()
}

