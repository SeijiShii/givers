package handler

import (
	"encoding/json"
	"log"
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
	log.Printf("[AUTH] Me: request from %s, Origin=%s", r.RemoteAddr, r.Header.Get("Origin"))
	log.Printf("[AUTH] Me: all cookies: %v", r.Header.Get("Cookie"))

	cookie, err := r.Cookie(auth.SessionCookieName())
	if err != nil {
		log.Printf("[AUTH] Me: FAIL — no %s cookie found (err=%v)", auth.SessionCookieName(), err)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	log.Printf("[AUTH] Me: cookie found — token=%s... (length=%d)", cookie.Value[:min(16, len(cookie.Value))], len(cookie.Value))

	userID, err := h.sv.ValidateSession(r.Context(), cookie.Value)
	if err != nil {
		log.Printf("[AUTH] Me: FAIL — session validation error: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_session"})
		return
	}
	log.Printf("[AUTH] Me: session valid — userID=%s", userID)

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		log.Printf("[AUTH] Me: FAIL — user not found: userID=%s, err=%v", userID, err)
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

	log.Printf("[AUTH] Me: SUCCESS — user=%s, email=%s, name=%s, role=%s", user.ID, user.Email, user.Name, resp.Role)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
