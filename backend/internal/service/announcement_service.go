package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// AnnouncementService はお知らせに関するビジネスロジックのインターフェース
type AnnouncementService interface {
	// Create は新しいお知らせを作成する
	Create(ctx context.Context, a *model.Announcement) error
	// GetByID は ID でお知らせを取得する
	GetByID(ctx context.Context, id string) (*model.Announcement, error)
	// Update はお知らせを更新する
	Update(ctx context.Context, a *model.Announcement) error
	// SetVisibility は公開/非表示を切り替える
	SetVisibility(ctx context.Context, id string, visible bool) error
	// ListVisible は公開中のお知らせ一覧を返す。userID があれば既読フラグを付与。
	ListVisible(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error)
	// ListAll は全お知らせを返す（管理画面用）
	ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	// MarkRead はお知らせを既読にする
	MarkRead(ctx context.Context, userID, announcementID string) error
	// UnreadCount は未読件数を返す
	UnreadCount(ctx context.Context, userID string) (int, error)
}
