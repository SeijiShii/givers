package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// MessageRepository はメッセージスレッド・メッセージの永続化インターフェース
type MessageRepository interface {
	// CreateThread は新しいスレッドを作成する
	CreateThread(ctx context.Context, t *model.MessageThread) error
	// GetThreadByID は ID でスレッドを取得する
	GetThreadByID(ctx context.Context, id string) (*model.MessageThread, error)
	// SetThreadActive はスレッドの活性/不活性を切り替える
	SetThreadActive(ctx context.Context, id string, active bool) error
	// ListThreadsForUser はユーザーが参加しているスレッド一覧を返す（未読数・最新メッセージ付き）
	ListThreadsForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	// ListThreadsForHost はホストが作成したスレッド一覧を返す（未読数・最新メッセージ付き）
	ListThreadsForHost(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	// InsertMessage はメッセージを挿入する
	InsertMessage(ctx context.Context, msg *model.Message) error
	// ListMessages はスレッド内のメッセージ一覧を返す（新しい順）
	ListMessages(ctx context.Context, threadID string, limit int, cursor string) ([]*model.Message, string, error)
	// MarkThreadRead はスレッドを既読にする
	MarkThreadRead(ctx context.Context, userID, threadID string) error
	// UnreadThreadCount はユーザーの未読スレッド数を返す
	UnreadThreadCount(ctx context.Context, userID string) (int, error)
}
