package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/pkg/auth"
)

// MeHandler は現在のユーザー情報を返すハンドラ
type MeHandler struct {
	userRepo   repository.UserRepository
	sv         auth.SessionValidator
	hostEmails map[string]bool
}

// NewMeHandler は MeHandler を生成する（DI: UserRepository を注入）
func NewMeHandler(userRepo repository.UserRepository, sv auth.SessionValidator, hostEmails []string) *MeHandler {
	set := make(map[string]bool, len(hostEmails))
	for _, e := range hostEmails {
		set[e] = true
	}
	return &MeHandler{userRepo: userRepo, sv: sv, hostEmails: set}
}

// meResponse は GET /api/me のレスポンス（User + role + suspended bool）
type meResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        string     `json:"role,omitempty"`
	Suspended   bool       `json:"suspended,omitempty"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Me は GET /api/me を処理する
func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	slog.Debug("me: request", "remote_addr", r.RemoteAddr, "origin", r.Header.Get("Origin"))

	cookie, err := r.Cookie(auth.SessionCookieName())
	if err != nil {
		slog.Warn("me: no session cookie", "cookie_name", auth.SessionCookieName(), "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	slog.Debug("me: cookie found", "token_prefix", cookie.Value[:min(16, len(cookie.Value))], "token_length", len(cookie.Value))

	userID, err := h.sv.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		slog.Warn("me: session validation failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_session"})
		return
	}
	slog.Debug("me: session valid", "user_id", userID)

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		slog.Warn("me: user not found", "user_id", userID, "error", err)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_not_found"})
		return
	}

	resp := meResponse{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		SuspendedAt: user.SuspendedAt,
		Suspended:   user.IsSuspended(),
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	if h.hostEmails[user.Email] {
		resp.Role = "host"
	}

	slog.Debug("me: success", "user_id", user.ID, "email", user.Email, "role", resp.Role)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
