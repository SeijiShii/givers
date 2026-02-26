package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/givers/backend/internal/logging"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: migrate [command]

Commands:
  (default)   差分マイグレーションを適用
  reset       全テーブルを DROP し、集約スキーマで再作成
  fresh       全テーブルを DROP し、全マイグレーションを順番に適用`)
	os.Exit(1)
}

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	logging.Setup()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://givers:givers@localhost:5432/givers?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logging.Fatal("connect failed", "error", err)
	}
	defer pool.Close()

	migrationDir := findMigrationDir()

	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "":
		runIncremental(ctx, pool, migrationDir)
	case "reset":
		runDropAll(ctx, pool, migrationDir)
		runConsolidated(ctx, pool, migrationDir)
	case "fresh":
		runDropAll(ctx, pool, migrationDir)
		runIncremental(ctx, pool, migrationDir)
	default:
		usage()
	}
}

func findMigrationDir() string {
	dir := "migrations"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		dir = "../migrations"
	}
	return dir
}

// collectUpFiles は .up.sql ファイル名をソート済みで返す
func collectUpFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logging.Fatal("read migrations dir failed", "error", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

func ensureSchemaMigrations(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
}

// ---------------------------------------------------------------------------
// (default) 差分マイグレーション
// ---------------------------------------------------------------------------
func runIncremental(ctx context.Context, pool *pgxpool.Pool, dir string) {
	ensureSchemaMigrations(ctx, pool)

	upFiles := collectUpFiles(dir)
	applied := 0
	for i, filename := range upFiles {
		name := strings.TrimSuffix(filename, ".up.sql")

		var exists bool
		_ = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name).Scan(&exists)
		if exists {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			logging.Fatal("read migration failed", "migration", name, "error", err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			logging.Fatal("migration failed", "migration", name, "error", err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
			logging.Fatal("record migration failed", "migration", name, "error", err)
		}
		applied++
		slog.Info("migration completed", "number", i+1, "migration", name)
	}

	if applied == 0 {
		slog.Info("all migrations already applied")
	} else {
		slog.Info("migrations completed", "count", applied)
	}
}

// ---------------------------------------------------------------------------
// 全テーブル DROP
// ---------------------------------------------------------------------------
func runDropAll(ctx context.Context, pool *pgxpool.Pool, dir string) {
	slog.Info("dropping all tables")
	sql, err := os.ReadFile(filepath.Join(dir, "000_drop_all.sql"))
	if err != nil {
		logging.Fatal("read 000_drop_all.sql failed", "error", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		logging.Fatal("drop all failed", "error", err)
	}
	slog.Info("all tables dropped")
}

// ---------------------------------------------------------------------------
// 集約スキーマで再作成
// ---------------------------------------------------------------------------
func runConsolidated(ctx context.Context, pool *pgxpool.Pool, dir string) {
	slog.Info("applying consolidated schema")
	sql, err := os.ReadFile(filepath.Join(dir, "000_consolidated.sql"))
	if err != nil {
		logging.Fatal("read 000_consolidated.sql failed", "error", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		logging.Fatal("consolidated apply failed", "error", err)
	}

	// 全マイグレーションを適用済みとして記録
	ensureSchemaMigrations(ctx, pool)
	upFiles := collectUpFiles(dir)
	for _, filename := range upFiles {
		name := strings.TrimSuffix(filename, ".up.sql")
		_, _ = pool.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT DO NOTHING", name)
	}
	slog.Info("consolidated schema applied", "migrations_marked", len(upFiles))
}
