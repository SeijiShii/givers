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
	pkgstripe "github.com/givers/backend/pkg/stripe"
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
	watchRepo := repository.NewPgWatchRepository(pool)
	projectUpdateRepo := repository.NewPgProjectUpdateRepository(pool)
	platformHealthRepo := repository.NewPgPlatformHealthRepository(pool)
	donationRepo := repository.NewPgDonationRepository(pool)
	costPresetRepo := repository.NewPgCostPresetRepository(pool)
	authService := service.NewAuthService(userRepo)
	projectService := service.NewProjectService(projectRepo)
	contactService := service.NewContactService(contactRepo)
	watchService := service.NewWatchService(watchRepo)
	projectUpdateService := service.NewProjectUpdateService(projectUpdateRepo)
	platformHealthService := service.NewPlatformHealthService(platformHealthRepo)
	adminUserService := service.NewAdminUserService(userRepo)
	donationService := service.NewDonationService(donationRepo)
	costPresetService := service.NewCostPresetService(costPresetRepo)

	// Stripe 設定（未設定の場合は Stripe 機能を無効化）
	stripeClient := pkgstripe.NewClient(
		os.Getenv("STRIPE_SECRET_KEY"),
		os.Getenv("STRIPE_CONNECT_CLIENT_ID"),
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	stripeService := service.NewStripeService(stripeClient, projectRepo, donationRepo, frontendURL)

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
	// Stripe が設定されている場合のみ Connect URL を生成する関数を渡す
	var connectURLFunc func(string) string
	if os.Getenv("STRIPE_CONNECT_CLIENT_ID") != "" {
		connectURLFunc = stripeService.GenerateConnectURL
	}
	stripeHandler := handler.NewStripeHandler(stripeService, frontendURL, sessionSecretBytes)
	projectHandler := handler.NewProjectHandler(projectService, connectURLFunc)
	contactHandler := handler.NewContactHandler(contactService)
	legalHandler := handler.NewLegalHandler(handler.LegalConfig{DocsDir: legalDocsDir})
	watchHandler := handler.NewWatchHandler(watchService)
	updateHandler := handler.NewProjectUpdateHandler(projectUpdateService, projectService)
	hostHandler := handler.NewHostHandler(platformHealthService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService, projectService)
	donationHandler := handler.NewDonationHandler(donationService)
	costPresetHandler := handler.NewCostPresetHandler(costPresetService)

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
	mux.Handle("DELETE /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Delete)))
	mux.Handle("PATCH /api/projects/{id}/status", wrapAuth(http.HandlerFunc(projectHandler.PatchStatus)))

	// プロジェクト更新 API
	mux.Handle("GET /api/projects/{id}/updates", http.HandlerFunc(updateHandler.List))
	mux.Handle("POST /api/projects/{id}/updates", wrapAuth(http.HandlerFunc(updateHandler.Create)))
	mux.Handle("PUT /api/projects/{id}/updates/{uid}", wrapAuth(http.HandlerFunc(updateHandler.UpdateUpdate)))
	mux.Handle("DELETE /api/projects/{id}/updates/{uid}", wrapAuth(http.HandlerFunc(updateHandler.Delete)))

	// ウォッチ API（認証必須）
	mux.Handle("POST /api/projects/{id}/watch", wrapAuth(http.HandlerFunc(watchHandler.Watch)))
	mux.Handle("DELETE /api/projects/{id}/watch", wrapAuth(http.HandlerFunc(watchHandler.Unwatch)))
	mux.Handle("GET /api/me/watches", wrapAuth(http.HandlerFunc(watchHandler.ListWatches)))

	// Admin routes (host-only — handler enforces IsHostFromContext)
	mux.Handle("GET /api/admin/contacts", wrapAuth(http.HandlerFunc(contactHandler.AdminList)))
	mux.Handle("PATCH /api/admin/contacts/{id}/status", wrapAuth(http.HandlerFunc(contactHandler.UpdateStatus)))
	mux.Handle("GET /api/admin/users", wrapAuth(http.HandlerFunc(adminUserHandler.List)))
	mux.Handle("PATCH /api/admin/users/{id}/suspend", wrapAuth(http.HandlerFunc(adminUserHandler.Suspend)))
	mux.Handle("GET /api/admin/disclosure-export", wrapAuth(http.HandlerFunc(adminUserHandler.DisclosureExport)))

	// Platform health (no auth required)
	mux.HandleFunc("GET /api/host", hostHandler.Get)

	// Donation routes (auth required)
	mux.Handle("GET /api/me/donations", wrapAuth(http.HandlerFunc(donationHandler.List)))
	mux.Handle("PATCH /api/me/donations/{id}", wrapAuth(http.HandlerFunc(donationHandler.Patch)))
	mux.Handle("DELETE /api/me/donations/{id}", wrapAuth(http.HandlerFunc(donationHandler.Delete)))
	mux.Handle("POST /api/me/migrate-from-token", wrapAuth(http.HandlerFunc(donationHandler.MigrateFromToken)))

	// Cost preset routes (auth required)
	mux.Handle("GET /api/me/cost-presets", wrapAuth(http.HandlerFunc(costPresetHandler.List)))
	mux.Handle("POST /api/me/cost-presets", wrapAuth(http.HandlerFunc(costPresetHandler.Create)))
	mux.Handle("PUT /api/me/cost-presets/reorder", wrapAuth(http.HandlerFunc(costPresetHandler.Reorder)))
	mux.Handle("PUT /api/me/cost-presets/{id}", wrapAuth(http.HandlerFunc(costPresetHandler.Update)))
	mux.Handle("DELETE /api/me/cost-presets/{id}", wrapAuth(http.HandlerFunc(costPresetHandler.Delete)))

	// Stripe routes (no auth — Stripe handles security via signatures/state)
	mux.HandleFunc("GET /api/stripe/connect/callback", stripeHandler.ConnectCallback)
	mux.HandleFunc("POST /api/donations/checkout", stripeHandler.Checkout)
	mux.HandleFunc("POST /api/webhooks/stripe", stripeHandler.Webhook)

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
