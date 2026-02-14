package handler

import (
	"encoding/json"
	"net/http"

	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/pkg/auth"
)

// MeHandler は現在のユーザー情報を返すハンドラ
type MeHandler struct {
	userRepo      repository.UserRepository
	sessionSecret []byte
}

// NewMeHandler は MeHandler を生成する（DI: UserRepository を注入）
func NewMeHandler(userRepo repository.UserRepository, sessionSecret []byte) *MeHandler {
	return &MeHandler{userRepo: userRepo, sessionSecret: sessionSecret}
}

// Me は GET /api/me を処理する
func (h *MeHandler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := auth.VerifySessionToken(cookie.Value, h.sessionSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_session"})
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "user_not_found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}
