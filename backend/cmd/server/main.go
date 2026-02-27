package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/givers/backend/internal/handler"
	"github.com/givers/backend/internal/logging"
	"github.com/givers/backend/internal/repository"
	"github.com/givers/backend/internal/service"
	"github.com/givers/backend/internal/storage"
	"github.com/givers/backend/pkg/auth"
	pkgstripe "github.com/givers/backend/pkg/stripe"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()    // backend/.env or CWD
	_ = godotenv.Load("../.env") // project root
	logging.Setup()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://givers:givers@localhost:5432/givers?sslmode=disable"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4321"
	}

	pool, err := repository.NewPool(context.Background(), dbURL)
	if err != nil {
		logging.Fatal("failed to connect to database", "error", err)
	}
	defer pool.Close()

	userRepo := repository.NewPgUserRepository(pool)
	projectRepo := repository.NewPgProjectRepository(pool)
	contactRepo := repository.NewPgContactRepository(pool)
	watchRepo := repository.NewPgWatchRepository(pool)
	projectUpdateRepo := repository.NewPgProjectUpdateRepository(pool)
	platformHealthRepo := repository.NewPgPlatformHealthRepository(pool)
	donationRepo := repository.NewPgDonationRepository(pool)
	activityRepo := repository.NewPgActivityRepository(pool)
	costPresetRepo := repository.NewPgCostPresetRepository(pool)
	sessionRepo := repository.NewPgSessionRepository(pool)

	authService := service.NewAuthService(userRepo)
	projectService := service.NewProjectService(projectRepo)
	contactService := service.NewContactService(contactRepo)
	watchService := service.NewWatchService(watchRepo)
	projectUpdateService := service.NewProjectUpdateService(projectUpdateRepo)
	platformHealthService := service.NewPlatformHealthService(platformHealthRepo)
	sessionSvc := service.NewSessionService(sessionRepo)
	adminUserService := service.NewAdminUserServiceWithSessions(userRepo, sessionRepo)
	// Stripe 設定（未設定の場合は Stripe 機能を無効化）
	stripeClient := pkgstripe.NewClient(
		os.Getenv("STRIPE_SECRET_KEY"),
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	activityService := service.NewActivityService(activityRepo)
	milestoneService := service.NewMilestoneService(projectRepo, donationRepo, activityRepo)
	stripeService := service.NewStripeServiceWithActivity(stripeClient, projectRepo, donationRepo, frontendURL, activityRepo, milestoneService)
	donationService := service.NewDonationService(donationRepo, stripeClient)
	costPresetService := service.NewCostPresetService(costPresetRepo)

	authRequired := os.Getenv("AUTH_REQUIRED") == "true"
	hostEmails := auth.ParseHostEmails(os.Getenv("HOST_EMAILS"))

	legalDocsDir := os.Getenv("LEGAL_DOCS_DIR")
	if legalDocsDir == "" {
		legalDocsDir = "./legal"
	}

	h := handler.New(pool, frontendURL)
	authHandler := handler.NewAuthHandler(authService, handler.AuthConfig{
		GoogleClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:  os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		GoogleRedirectPath:  "/api/auth/google/callback",
		GitHubRedirectPath:  "/api/auth/github/callback",
		DiscordRedirectPath: "/api/auth/discord/callback",
		FrontendURL:         frontendURL,
	}, sessionSvc)
	providersHandler := handler.NewProvidersHandler(handler.ProvidersConfig{
		GitHubClientID:  os.Getenv("GITHUB_CLIENT_ID"),
		DiscordClientID: os.Getenv("DISCORD_CLIENT_ID"),
		AppleClientID:   os.Getenv("APPLE_CLIENT_ID"),
		EnableEmail:     os.Getenv("ENABLE_EMAIL_LOGIN") == "true",
	})
	meHandler := handler.NewMeHandler(userRepo, sessionSvc, hostEmails)
	// Stripe が設定されている場合のみ v2 API でアカウント作成+オンボーディングを行う
	var connectAccountFunc handler.ConnectAccountFunc
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		connectAccountFunc = stripeService.CreateAccountAndOnboarding
	}
	stripeHandler := handler.NewStripeHandler(stripeService, frontendURL, sessionSvc)
	projectHandler := handler.NewProjectHandlerWithActivity(projectService, connectAccountFunc, activityService)
	contactHandler := handler.NewContactHandler(contactService)
	legalHandler := handler.NewLegalHandler(handler.LegalConfig{DocsDir: legalDocsDir})
	watchHandler := handler.NewWatchHandler(watchService)
	updateHandler := handler.NewProjectUpdateHandler(projectUpdateService, projectService)
	hostHandler := handler.NewHostHandler(platformHealthService)
	adminUserHandler := handler.NewAdminUserHandler(adminUserService, projectService, donationRepo)
	donationHandler := handler.NewDonationHandler(donationService)
	activityHandler := handler.NewActivityHandler(activityService)
	chartHandler := handler.NewChartHandler(projectService, donationRepo)
	costPresetHandler := handler.NewCostPresetHandler(costPresetService)

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}
	imageStorage := storage.NewLocalStorage(uploadsDir, "/uploads")
	imageHandler := handler.NewImageHandler(imageStorage, projectService, projectRepo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/auth/providers", providersHandler.Providers)
	mux.HandleFunc("GET /api/auth/google/login", authHandler.GoogleLoginURL)
	mux.HandleFunc("GET /api/auth/google/callback", authHandler.GoogleCallback)
	mux.HandleFunc("GET /api/auth/github/login", authHandler.GitHubLoginURL)
	mux.HandleFunc("GET /api/auth/github/callback", authHandler.GitHubCallback)
	mux.HandleFunc("GET /api/auth/discord/login", authHandler.DiscordLoginURL)
	mux.HandleFunc("GET /api/auth/discord/callback", authHandler.DiscordCallback)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/finalize", authHandler.FinalizeLogin)
	mux.HandleFunc("GET /api/me", meHandler.Me)
	mux.HandleFunc("POST /api/contact", contactHandler.Submit)
	mux.HandleFunc("GET /api/legal/{type}", legalHandler.Legal)

	// プロジェクト API（一覧・詳細は認証不要）
	mux.Handle("GET /api/projects", http.HandlerFunc(projectHandler.List))
	mux.Handle("GET /api/projects/{id}", http.HandlerFunc(projectHandler.Get))

	// 認証必要エンドポイント
	hostMW := auth.HostMiddleware(hostEmails, func(ctx context.Context, userID string) (string, error) {
		u, err := userRepo.FindByID(ctx, userID)
		if err != nil {
			return "", err
		}
		return u.Email, nil
	})
	wrapAuth := func(next http.Handler) http.Handler {
		if authRequired {
			return auth.RequireAuth(sessionSvc)(hostMW(next))
		}
		return auth.DevAuth(hostMW(next))
	}
	mux.Handle("GET /api/me/projects", wrapAuth(http.HandlerFunc(projectHandler.MyProjects)))
	mux.Handle("POST /api/projects", wrapAuth(http.HandlerFunc(projectHandler.Create)))
	mux.Handle("PUT /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Update)))
	mux.Handle("DELETE /api/projects/{id}", wrapAuth(http.HandlerFunc(projectHandler.Delete)))
	mux.Handle("PATCH /api/projects/{id}/status", wrapAuth(http.HandlerFunc(projectHandler.PatchStatus)))
	mux.Handle("POST /api/projects/{id}/image", wrapAuth(http.HandlerFunc(imageHandler.Upload)))
	mux.Handle("DELETE /api/projects/{id}/image", wrapAuth(http.HandlerFunc(imageHandler.Delete)))

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

	// Activity feed & chart (no auth required)
	mux.HandleFunc("GET /api/activity", activityHandler.GlobalFeed)
	mux.HandleFunc("GET /api/projects/{id}/activity", activityHandler.ProjectFeed)
	mux.HandleFunc("GET /api/projects/{id}/chart", chartHandler.Chart)

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
	mux.HandleFunc("GET /api/stripe/onboarding/return", stripeHandler.OnboardingReturn)
	mux.HandleFunc("GET /api/stripe/onboarding/refresh", stripeHandler.OnboardingRefresh)
	// Rate limit on checkout endpoint (10 req/min per IP)
	checkoutRL := handler.NewRateLimiter(10)
	mux.Handle("POST /api/donations/checkout", checkoutRL.Middleware(http.HandlerFunc(stripeHandler.Checkout)))
	mux.HandleFunc("POST /api/webhooks/stripe", stripeHandler.Webhook)

	// 画像ファイルの静的配信
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Middleware chain: RequestLogger → SecurityHeaders → CORS → mux
	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler.RequestLogger(handler.SecurityHeaders(h.CORS(mux))),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
