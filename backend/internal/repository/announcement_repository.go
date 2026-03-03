package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// AnnouncementRepository はお知らせの永続化インターフェース
type AnnouncementRepository interface {
	// Insert は新しいお知らせを作成する
	Insert(ctx context.Context, a *model.Announcement) error
	// GetByID は ID でお知らせを取得する
	GetByID(ctx context.Context, id string) (*model.Announcement, error)
	// Update はお知らせを更新する
	Update(ctx context.Context, a *model.Announcement) error
	// SetVisibility は公開/非表示を切り替える
	SetVisibility(ctx context.Context, id string, visible bool) error
	// ListVisible は公開中（visible=true かつ published_at <= now）のお知らせを返す
	ListVisible(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	// ListAll は全お知らせ（非表示・予約中含む）を返す（管理画面用）
	ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	// MarkRead はユーザーのお知らせ既読をマークする
	MarkRead(ctx context.Context, userID, announcementID string) error
	// UnreadCount は公開中お知らせのうちユーザーが未読の件数を返す
	UnreadCount(ctx context.Context, userID string) (int, error)
	// SetReadFlags は announcements スライスの IsRead フィールドをセットする
	SetReadFlags(ctx context.Context, userID string, announcements []*model.Announcement) error
}
