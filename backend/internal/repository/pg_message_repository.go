package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgMessageRepository は MessageRepository の PostgreSQL 実装
type PgMessageRepository struct {
	pool *pgxpool.Pool
}

// NewPgMessageRepository は PgMessageRepository を生成する
func NewPgMessageRepository(pool *pgxpool.Pool) *PgMessageRepository {
	return &PgMessageRepository{pool: pool}
}

func (r *PgMessageRepository) CreateThread(ctx context.Context, t *model.MessageThread) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO message_threads (host_user_id, participant_user_id, subject, active)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		t.HostUserID, t.ParticipantUserID, t.Subject, t.Active,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *PgMessageRepository) GetThreadByID(ctx context.Context, id string) (*model.MessageThread, error) {
	var t model.MessageThread
	err := r.pool.QueryRow(ctx,
		`SELECT id, host_user_id, participant_user_id, subject, active, created_at, updated_at
		 FROM message_threads WHERE id = $1`, id,
	).Scan(&t.ID, &t.HostUserID, &t.ParticipantUserID, &t.Subject, &t.Active,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return &t, nil
}

func (r *PgMessageRepository) SetThreadActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE message_threads SET active = $1, updated_at = NOW() WHERE id = $2`,
		active, id,
	)
	return err
}

func (r *PgMessageRepository) ListThreadsForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	query := `SELECT t.id, t.host_user_id, t.participant_user_id, t.subject, t.active,
		 t.created_at, t.updated_at,
		 COALESCE(hu.name, '') AS host_name,
		 COALESCE(pu.name, '') AS participant_name,
		 COALESCE(lm.body, '') AS last_message_body,
		 COALESCE(lm.created_at::text, '') AS last_message_at,
		 (SELECT COUNT(*) FROM messages m2
		  WHERE m2.thread_id = t.id
		    AND m2.created_at > COALESCE(
		      (SELECT mr.last_read_at FROM message_reads mr WHERE mr.user_id = $1 AND mr.thread_id = t.id),
		      '1970-01-01'::timestamptz
		    )
		 ) AS unread_count
		 FROM message_threads t
		 LEFT JOIN users hu ON hu.id = t.host_user_id
		 LEFT JOIN users pu ON pu.id = t.participant_user_id
		 LEFT JOIN LATERAL (
		   SELECT body, created_at FROM messages
		   WHERE thread_id = t.id ORDER BY created_at DESC LIMIT 1
		 ) lm ON true
		 WHERE (t.host_user_id = $1 OR t.participant_user_id = $1)`
	args := []any{userID}
	argIdx := 2

	if cursor != "" {
		query += ` AND t.updated_at < (SELECT updated_at FROM message_threads WHERE id = $` + itoa(argIdx) + `)`
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY t.updated_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit+1)

	return r.scanThreadList(ctx, query, args, limit)
}

func (r *PgMessageRepository) ListThreadsForHost(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error) {
	query := `SELECT t.id, t.host_user_id, t.participant_user_id, t.subject, t.active,
		 t.created_at, t.updated_at,
		 COALESCE(hu.name, '') AS host_name,
		 COALESCE(pu.name, '') AS participant_name,
		 COALESCE(lm.body, '') AS last_message_body,
		 COALESCE(lm.created_at::text, '') AS last_message_at,
		 (SELECT COUNT(*) FROM messages m2
		  WHERE m2.thread_id = t.id
		    AND m2.created_at > COALESCE(
		      (SELECT mr.last_read_at FROM message_reads mr WHERE mr.user_id = $1 AND mr.thread_id = t.id),
		      '1970-01-01'::timestamptz
		    )
		 ) AS unread_count
		 FROM message_threads t
		 LEFT JOIN users hu ON hu.id = t.host_user_id
		 LEFT JOIN users pu ON pu.id = t.participant_user_id
		 LEFT JOIN LATERAL (
		   SELECT body, created_at FROM messages
		   WHERE thread_id = t.id ORDER BY created_at DESC LIMIT 1
		 ) lm ON true
		 WHERE t.host_user_id = $1`
	args := []any{hostUserID}
	argIdx := 2

	if cursor != "" {
		query += ` AND t.updated_at < (SELECT updated_at FROM message_threads WHERE id = $` + itoa(argIdx) + `)`
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY t.updated_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit+1)

	return r.scanThreadList(ctx, query, args, limit)
}

func (r *PgMessageRepository) scanThreadList(ctx context.Context, query string, args []any, limit int) ([]*model.MessageThread, string, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var threads []*model.MessageThread
	for rows.Next() {
		var t model.MessageThread
		if err := rows.Scan(&t.ID, &t.HostUserID, &t.ParticipantUserID, &t.Subject, &t.Active,
			&t.CreatedAt, &t.UpdatedAt,
			&t.HostName, &t.ParticipantName,
			&t.LastMessageBody, &t.LastMessageAt,
			&t.UnreadCount); err != nil {
			return nil, "", err
		}
		threads = append(threads, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(threads) > limit {
		nextCursor = threads[limit-1].ID
		threads = threads[:limit]
	}

	return threads, nextCursor, nil
}

func (r *PgMessageRepository) InsertMessage(ctx context.Context, msg *model.Message) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO messages (thread_id, sender_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		msg.ThreadID, msg.SenderID, msg.Body,
	).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return err
	}
	// スレッドの updated_at を更新
	_, err = r.pool.Exec(ctx,
		`UPDATE message_threads SET updated_at = NOW() WHERE id = $1`,
		msg.ThreadID,
	)
	return err
}

func (r *PgMessageRepository) ListMessages(ctx context.Context, threadID string, limit int, cursor string) ([]*model.Message, string, error) {
	query := `SELECT m.id, m.thread_id, m.sender_id, m.body, m.created_at,
		 COALESCE(u.name, '匿名') AS sender_name
		 FROM messages m
		 LEFT JOIN users u ON u.id = m.sender_id
		 WHERE m.thread_id = $1`
	args := []any{threadID}
	argIdx := 2

	if cursor != "" {
		query += ` AND m.created_at < (SELECT created_at FROM messages WHERE id = $` + itoa(argIdx) + `)`
		args = append(args, cursor)
		argIdx++
	}

	query += ` ORDER BY m.created_at DESC LIMIT $` + itoa(argIdx)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.SenderID, &m.Body, &m.CreatedAt, &m.SenderName); err != nil {
			return nil, "", err
		}
		messages = append(messages, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(messages) > limit {
		nextCursor = messages[limit-1].ID
		messages = messages[:limit]
	}

	return messages, nextCursor, nil
}

func (r *PgMessageRepository) MarkThreadRead(ctx context.Context, userID, threadID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO message_reads (user_id, thread_id, last_read_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id, thread_id) DO UPDATE SET last_read_at = NOW()`,
		userID, threadID,
	)
	return err
}

func (r *PgMessageRepository) UnreadThreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_threads t
		 WHERE (t.host_user_id = $1 OR t.participant_user_id = $1)
		   AND EXISTS (
		     SELECT 1 FROM messages m
		     WHERE m.thread_id = t.id
		       AND m.created_at > COALESCE(
		         (SELECT mr.last_read_at FROM message_reads mr WHERE mr.user_id = $1 AND mr.thread_id = t.id),
		         '1970-01-01'::timestamptz
		       )
		   )`, userID,
	).Scan(&count)
	return count, err
}
