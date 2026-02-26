package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage はローカルファイルシステムに画像を保存する Storage 実装。
type LocalStorage struct {
	baseDir   string // ディスク上のルートディレクトリ (例: "./uploads")
	urlPrefix string // HTTP で配信する際の URL プレフィックス (例: "/uploads")
}

// NewLocalStorage は LocalStorage を生成する。
func NewLocalStorage(baseDir, urlPrefix string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir, urlPrefix: urlPrefix}
}

func (s *LocalStorage) Save(_ context.Context, key string, data io.Reader, _ string) (string, error) {
	dest := filepath.Join(s.baseDir, key)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("storage: mkdir: %w", err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("storage: create: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("storage: write: %w", err)
	}

	url := s.urlPrefix + "/" + key
	return url, nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	dest := filepath.Join(s.baseDir, key)
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: remove: %w", err)
	}
	return nil
}
