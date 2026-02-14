package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

// CreateSessionToken はユーザーIDから署名付きセッショントークンを生成する
func CreateSessionToken(userID string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(userID))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.URLEncoding.EncodeToString([]byte(userID)) + "." + sig
}

// VerifySessionToken はトークンを検証しユーザーIDを返す
func VerifySessionToken(token string, secret []byte) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid token format")
	}
	payload, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	userID := string(payload)

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", errors.New("invalid signature")
	}
	return userID, nil
}

const sessionCookieName = "givers_session"
const minSecretLen = 32

// SessionCookieName はセッションクッキー名
func SessionCookieName() string {
	return sessionCookieName
}

// SessionSecretBytes は文字列からセッション署名用のバイト列を生成する（最低32バイト）
func SessionSecretBytes(s string) []byte {
	b := []byte(s)
	if len(b) < minSecretLen {
		out := make([]byte, minSecretLen)
		copy(out, b)
		return out
	}
	return b
}
