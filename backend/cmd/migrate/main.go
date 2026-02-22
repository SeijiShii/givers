package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()       // backend/.env or CWD
	_ = godotenv.Load("../.env") // project root

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://givers:givers@localhost:5432/givers?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// migrations ディレクトリから .up.sql ファイルを自動検出
	migrationDir := "migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		migrationDir = "../migrations"
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}

	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	// schema_migrations テーブルで適用済みを管理
	_, _ = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	applied := 0
	for i, filename := range upFiles {
		name := strings.TrimSuffix(filename, ".up.sql")

		// 適用済みチェック
		var exists bool
		_ = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)", name).Scan(&exists)
		if exists {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationDir, filename))
		if err != nil {
			log.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			log.Fatalf("migrate %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
			log.Fatalf("record migration %s: %v", name, err)
		}
		applied++
		log.Printf("migration %d: %s completed", i+1, name)
	}

	if applied == 0 {
		log.Println("all migrations already applied")
	} else {
		log.Printf("%d migrations completed", applied)
	}
}
