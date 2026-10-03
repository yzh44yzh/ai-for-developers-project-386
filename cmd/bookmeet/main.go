package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yzh44yzh/bookmeet/internal/api"
	"github.com/yzh44yzh/bookmeet/internal/postgres"
	"github.com/yzh44yzh/bookmeet/internal/web"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Local development default; dev-only credentials.
		dsn = "postgres://test:test@localhost:5432/testdb?sslmode=disable"
	}
	ctx := context.Background()

	if err := postgres.Migrate(dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	users := postgres.NewUsers(pool)
	webMux := web.NewMux(users)

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", webMux)
	mux.Handle("GET /login", webMux)
	mux.Handle("POST /login", webMux)
	mux.Handle("/", api.NewMux(users, postgres.NewMeetings(pool)))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
