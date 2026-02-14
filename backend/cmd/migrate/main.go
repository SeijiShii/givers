package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

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

	for i, name := range []string{"001_create_users", "002_add_github_id", "003_create_projects", "004_insert_dev_user", "005_add_owner_want_monthly"} {
		sql, err := os.ReadFile("migrations/" + name + ".up.sql")
		if err != nil {
			log.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			log.Fatalf("migrate %s: %v", name, err)
		}
		log.Printf("migration %d: %s completed", i+1, name)
	}
	log.Println("all migrations completed")
}
