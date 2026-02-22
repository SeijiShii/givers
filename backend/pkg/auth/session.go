package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const sessionCookieName = "givers_session"

// SessionMaxAge はセッション Cookie の有効期間（秒）
const SessionMaxAge = 60 * 60 * 24 * 7 // 7 days

// SessionDuration はセッションの有効期間
const SessionDuration = 7 * 24 * time.Hour

// SessionCookieName はセッションクッキー名を返す
func SessionCookieName() string {
	return sessionCookieName
}

// GenerateSessionToken は crypto/rand で 32 バイトのランダムトークンを生成し hex エンコードする
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
