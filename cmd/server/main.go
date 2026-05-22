package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/battle"
	"github.com/DevJoshBrown/BeatBattler/internal/battle_participants"
	"github.com/DevJoshBrown/BeatBattler/internal/config"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/scheduler"
	"github.com/DevJoshBrown/BeatBattler/internal/user"
	votes "github.com/DevJoshBrown/BeatBattler/internal/vote"
	"github.com/DevJoshBrown/BeatBattler/pkg/storage/postgres"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Load config (internal/config)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create pool (pkg/storage/postgres)
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create a connection pool")
	}
	log.Printf("database connected - connections: %v", pool.Stat().TotalConns())
	defer pool.Close()

	//create queries & handlers (internal/db - sqlc generated)
	queries := db.New(pool)
	userHandler := user.NewHandler(queries)
	sched := scheduler.NewScheduler(queries, pool)
	battleHandler := battle.NewHandler(queries, sched)
	participantHandler := battle_participants.NewHandler(queries)
	voteHandler := votes.NewHandler(queries)

	// Register routes on chi router
	r := chi.NewRouter()

	// Check server health
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handlers
	// users
	r.Post("/users", userHandler.CreateUser)
	r.Get("/users/{id}", userHandler.GetUser)
	// battles
	r.Post("/battles", battleHandler.CreateBattle)
	r.Get("/battles", battleHandler.ListBattles)
	r.Get("/battles/{id}", battleHandler.GetBattle)
	r.Post("/battles/{id}/start", battleHandler.StartBattle)
	r.Get("/battles/{id}/results", battleHandler.GetResults)
	// participants
	r.Post("/battles/{id}/join", participantHandler.CreateParticipant)
	r.Post("/battles/{id}/submit", participantHandler.SubmitParticipant)
	r.Get("/battles/{id}/participants", participantHandler.ListParticipants)
	// votes
	r.Post("/battles/{id}/vote", voteHandler.CastVote)
	r.Post("/battles/{id}/confirm-votes", voteHandler.ConfirmVotes)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("BeatBattler listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
