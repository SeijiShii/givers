package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/givers/backend/internal/handler"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/pkg/auth"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://givers:givers@localhost:5432/givers?sslmode=disable"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4321"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-change-in-production-32bytes"
	}

	pool, err := repository.NewPool(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewPgUserRepository(pool)
	projectRepo := repository.NewPgProjectRepository(pool)
	contactRepo := repository.NewPgContactRepository(pool)
	authService := service.NewAuthService(userRepo)
	projectService := service.NewProjectService(projectRepo)
	contactService := service.NewContactService(contactRepo)

	authRequired := os.Getenv("AUTH_REQUIRED") == "true"
	sessionSecretBytes := auth.SessionSecretBytes(sessionSecret)

	legalDocsDir := os.Getenv("LEGAL_DOCS_DIR")
	if legalDocsDir == "" {
		legalDocsDir = "./legal"
	}

	h := handler.New(pool, frontendURL)
	authHandler := handler.NewAuthHandler(authService, handler.AuthConfig{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GoogleRedirectPath: "/api/auth/google/callback",
		GitHubRedirectPath: "/api/auth/github/callback",
		SessionSecret:      sessionSecret,
		FrontendURL:        frontendURL,
	})
	providersHandler := handler.NewProvidersHandler(handler.ProvidersConfig{
		GitHubClientID: os.Getenv("GITHUB_CLIENT_ID"),
		AppleClientID:  os.Getenv("APPLE_CLIENT_ID"),
		EnableEmail:    os.Getenv("ENABLE_EMAIL_LOGIN") == "true",
	})
	meHandler := handler.NewMeHandler(userRepo, sessionSecretBytes)
	projectHandler := handler.NewProjectHandler(projectService)
	contactHandler := handler.NewContactHandler(contactService)
	legalHandler := handler.NewLegalHandler(handler.LegalConfig{DocsDir: legalDocsDir})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/auth/providers", providersHandler.Providers)
	mux.HandleFunc("GET /api/auth/google/login", authHandler.GoogleLoginURL)
	mux.HandleFunc("GET /api/auth/google/callback", authHandler.GoogleCallback)
	mux.HandleFunc("GET /api/auth/github/login", authHandler.GitHubLoginURL)
	mux.HandleFunc("GET /api/auth/github/callback", authHandler.GitHubCallback)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/me", meHandler.Me)
	mux.HandleFunc("POST /api/contact", contactHandler.Submit)
	mux.HandleFunc("GET /api/legal/{type}", legalHandler.Legal)

	// プロジェクト API（一覧・詳細は認証不要）
	mux.Handle("GET /api/projects", http.HandlerFunc(projectHandler.List))
	mux.Handle("GET /api/projects/{id}", http.HandlerFunc(projectHandler.Get))

	// 認証必要エンドポイント
	wrapAuth := func(next http.Handler) http.Handler {
		if authRequired {
			return auth.RequireAuth(sessionSecretBytes)(next)
		}
		return auth.DevAuth(next)
	}
	mux.Handle("GET /api/me/projects", wrapAuth(http.HandlerFunc(projectHandler.MyProjects)))
	mux.Handle("POST /api/projects", wrapAuth(http.HandlerFunc(projectHandler.Create)))
	mux.Handle("PUT /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Update)))

	// Admin routes (host-only — handler enforces IsHostFromContext)
	mux.Handle("GET /api/admin/contacts", wrapAuth(http.HandlerFunc(contactHandler.AdminList)))

	server := &http.Server{
		Addr:         ":8080",
		Handler:      h.CORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
