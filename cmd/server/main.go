package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/config"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/pkg/storage/postgres"
	"github.com/go-chi/chi/v5"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create a connection pool")
	}
	log.Printf("database connected - connections: %v", pool.Stat().TotalConns())
	defer pool.Close()

	queries := db.New(pool)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := ":" + cfg.Port
	log.Printf("BeatBattler listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
