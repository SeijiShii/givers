package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ConsentRepository はユーザー同意の永続化インターフェース
type ConsentRepository interface {
	// Upsert は同意記録を挿入または更新する（1ユーザー1レコード）
	Upsert(ctx context.Context, c *model.UserConsent) error
	// GetByUserID はユーザーIDで同意記録を取得する。未同意の場合は nil, nil を返す。
	GetByUserID(ctx context.Context, userID string) (*model.UserConsent, error)
}
