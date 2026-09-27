package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yzh44yzh/bookmeet/internal/hello"
	"github.com/yzh44yzh/bookmeet/internal/home"
	"github.com/yzh44yzh/bookmeet/internal/postgres"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/", home.Handler)
	mux.HandleFunc("GET /hello", hello.Handler)
	mux.HandleFunc("POST /hello", hello.BodyLogger)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
