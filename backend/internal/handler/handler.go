package handler

import (
	"net/http"

	"github.com/givers/backend/internal/repository"
)

// Handler は HTTP ハンドラを提供する（MVC の Controller 相当）
type Handler struct {
	db          repository.DB
	frontendURL string
}

// New は Handler を生成する（DI: DB インターフェースを注入）
func New(db repository.DB, frontendURL string) *Handler {
	return &Handler{db: db, frontendURL: frontendURL}
}

func (h *Handler) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", h.frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
