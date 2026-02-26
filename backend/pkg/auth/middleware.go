package auth

import (
	"context"
	"encoding/json"
	"log/slog"
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

// SessionValidator はセッショントークンを検証して userID を返すインターフェース
type SessionValidator interface {
	ValidateSession(ctx context.Context, token string) (string, error)
}

// RequireAuth は認証必須ミドルウェア。DB セッションを検証し、userID を context にセットする
func RequireAuth(sv SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Debug("auth check", "method", r.Method, "path", r.URL.Path)

			cookie, err := r.Cookie(SessionCookieName())
			if err != nil {
				slog.Warn("auth rejected: no session cookie", "cookie_name", SessionCookieName(), "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			tokenPrefix := cookie.Value
			if len(tokenPrefix) > 16 {
				tokenPrefix = tokenPrefix[:16]
			}
			slog.Debug("auth cookie found", "token_prefix", tokenPrefix, "token_length", len(cookie.Value))

			userID, err := sv.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				slog.Warn("auth rejected: validation failed", "error", err)
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_session"})
				return
			}

			slog.Debug("auth passed", "user_id", userID, "path", r.URL.Path)
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
