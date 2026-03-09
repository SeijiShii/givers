package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// MessageService はメッセージングに関するビジネスロジックのインターフェース
type MessageService interface {
	// CreateThread はホストが新しいスレッドを作成し初回メッセージを送信する
	CreateThread(ctx context.Context, hostUserID, participantUserID, subject, body string) (*model.MessageThread, error)
	// SendMessage はスレッドにメッセージを送信する（活性かつ参加者であることを検証）
	SendMessage(ctx context.Context, threadID, senderID, body string) (*model.Message, error)
	// GetThread はスレッドを取得する（参加者であることを検証）
	GetThread(ctx context.Context, threadID, requestingUserID string) (*model.MessageThread, error)
	// ListThreads はユーザーのスレッド一覧を返す
	ListThreads(ctx context.Context, userID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	// ListThreadsForHost はホストのスレッド一覧を返す（管理画面用）
	ListThreadsForHost(ctx context.Context, hostUserID string, limit int, cursor string) ([]*model.MessageThread, string, error)
	// ListMessages はスレッド内のメッセージ一覧を返す（閲覧時に既読マーク）
	ListMessages(ctx context.Context, threadID, requestingUserID string, limit int, cursor string) ([]*model.Message, string, error)
	// SetThreadActive はスレッドの活性/不活性を切り替える（ホストのみ）
	SetThreadActive(ctx context.Context, threadID, hostUserID string, active bool) error
	// UnreadCount はユーザーの未読スレッド数を返す
	UnreadCount(ctx context.Context, userID string) (int, error)
	// BroadcastThread はホストが全ユーザーにスレッドを一斉作成する。作成数を返す。
	BroadcastThread(ctx context.Context, hostUserID, subject, body string) (int, error)
}
