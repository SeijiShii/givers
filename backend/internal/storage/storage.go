package storage

import (
	"context"
	"io"
)

// Storage は画像ファイルの保存・削除を抽象化するインターフェース。
// ローカルファイルシステム実装の他、S3 / Cloudflare R2 等に差し替え可能。
type Storage interface {
	// Save はファイルを保存し、公開 URL を返す。
	// key はストレージ内の一意パス (例: "projects/<id>/<uuid>.jpg")。
	Save(ctx context.Context, key string, data io.Reader, contentType string) (url string, err error)

	// Delete は key に対応するファイルを削除する。
	Delete(ctx context.Context, key string) error
}
