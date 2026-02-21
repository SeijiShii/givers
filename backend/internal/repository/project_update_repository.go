package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ProjectUpdateRepository はプロジェクト更新の永続化インターフェース
type ProjectUpdateRepository interface {
	// ListByProjectID はプロジェクトに属する更新一覧を返す。
	// includeHidden=true の場合、visible=false の更新も含む。
	ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error)
	// GetByID は ID で更新を取得する
	GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error)
	// Create は新しい更新を作成する
	Create(ctx context.Context, update *model.ProjectUpdate) error
	// Update は title, body, visible, updated_at を更新する
	Update(ctx context.Context, update *model.ProjectUpdate) error
	// Delete は visible=false をセットするソフトデリート
	Delete(ctx context.Context, id string) error
}
