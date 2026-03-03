package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgAnnouncementRepository は AnnouncementRepository の PostgreSQL 実装
type PgAnnouncementRepository struct {
	pool *pgxpool.Pool
}

// NewPgAnnouncementRepository は PgAnnouncementRepository を生成する
func NewPgAnnouncementRepository(pool *pgxpool.Pool) *PgAnnouncementRepository {
	return &PgAnnouncementRepository{pool: pool}
}

func (r *PgAnnouncementRepository) Insert(ctx context.Context, a *model.Announcement) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO announcements (author_id, title, body, severity, visible, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		a.AuthorID, a.Title, a.Body, a.Severity, a.Visible, a.PublishedAt,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *PgAnnouncementRepository) GetByID(ctx context.Context, id string) (*model.Announcement, error) {
	var a model.Announcement
	err := r.pool.QueryRow(ctx,
		`SELECT id, author_id, title, body, severity, visible, published_at, created_at, updated_at
		 FROM announcements WHERE id = $1`, id,
	).Scan(&a.ID, &a.AuthorID, &a.Title, &a.Body, &a.Severity, &a.Visible,
		&a.PublishedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return &a, nil
}

func (r *PgAnnouncementRepository) Update(ctx context.Context, a *model.Announcement) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE announcements
		 SET title = $1, body = $2, severity = $3, published_at = $4, updated_at = NOW()
		 WHERE id = $5`,
		a.Title, a.Body, a.Severity, a.PublishedAt, a.ID,
	)
	return err
}

func (r *PgAnnouncementRepository) SetVisibility(ctx context.Context, id string, visible bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE announcements SET visible = $1, updated_at = NOW() WHERE id = $2`,
		visible, id,
	)
	return err
}

func (r *PgAnnouncementRepository) ListVisible(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	query := `SELECT id, author_id, title, body, severity, visible, published_at, created_at, updated_at
		 FROM announcements
		 WHERE visible = true AND published_at <= NOW()`
	args := []any{}
	argIdx := 1

	if cursor != "" {
		query += ` AND published_at < (SELECT published_at FROM announcements WHERE id = $` + itoa(argIdx) + `)`
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY published_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit+1) // fetch one extra to determine next_cursor

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var announcements []*model.Announcement
	for rows.Next() {
		var a model.Announcement
		if err := rows.Scan(&a.ID, &a.AuthorID, &a.Title, &a.Body, &a.Severity,
			&a.Visible, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, "", err
		}
		announcements = append(announcements, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(announcements) > limit {
		nextCursor = announcements[limit-1].ID
		announcements = announcements[:limit]
	}

	return announcements, nextCursor, nil
}

func (r *PgAnnouncementRepository) ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	query := `SELECT id, author_id, title, body, severity, visible, published_at, created_at, updated_at
		 FROM announcements WHERE 1=1`
	args := []any{}
	argIdx := 1

	if cursor != "" {
		query += ` AND created_at < (SELECT created_at FROM announcements WHERE id = $` + itoa(argIdx) + `)`
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var announcements []*model.Announcement
	for rows.Next() {
		var a model.Announcement
		if err := rows.Scan(&a.ID, &a.AuthorID, &a.Title, &a.Body, &a.Severity,
			&a.Visible, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, "", err
		}
		announcements = append(announcements, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(announcements) > limit {
		nextCursor = announcements[limit-1].ID
		announcements = announcements[:limit]
	}

	return announcements, nextCursor, nil
}

func (r *PgAnnouncementRepository) MarkRead(ctx context.Context, userID, announcementID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO announcement_reads (user_id, announcement_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, announcement_id) DO NOTHING`,
		userID, announcementID,
	)
	return err
}

func (r *PgAnnouncementRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM announcements a
		 WHERE a.visible = true AND a.published_at <= NOW()
		   AND NOT EXISTS (
		     SELECT 1 FROM announcement_reads ar
		     WHERE ar.announcement_id = a.id AND ar.user_id = $1
		   )`, userID,
	).Scan(&count)
	return count, err
}

func (r *PgAnnouncementRepository) SetReadFlags(ctx context.Context, userID string, announcements []*model.Announcement) error {
	if len(announcements) == 0 {
		return nil
	}

	ids := make([]string, len(announcements))
	for i, a := range announcements {
		ids[i] = a.ID
	}

	rows, err := r.pool.Query(ctx,
		`SELECT announcement_id FROM announcement_reads
		 WHERE user_id = $1 AND announcement_id = ANY($2)`,
		userID, ids,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	readSet := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		readSet[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, a := range announcements {
		isRead := readSet[a.ID]
		a.IsRead = &isRead
	}
	return nil
}

// itoa is declared in pg_contact_repository.go
