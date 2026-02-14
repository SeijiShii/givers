package auth

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "user_id"

// UserIDFromContext は context から userID を取得する
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

// WithUserID は context に userID をセットする
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// RequireAuth は認証必須ミドルウェア。セッションを検証し、userID を context にセットする
func RequireAuth(sessionSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName())
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			userID, err := VerifySessionToken(cookie.Value, sessionSecret)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_session"})
				return
			}

			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DevUserID は開発用のダミー userID（AUTH_REQUIRED=false 時に使用）
const DevUserID = "dev-user-id"

// DevAuth は開発用ミドルウェア。ダミー userID を context にセットする
func DevAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithUserID(r.Context(), DevUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
