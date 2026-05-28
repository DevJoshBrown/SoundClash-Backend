package main

import (
	"context"
	"log"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/audio"
	"github.com/DevJoshBrown/BeatBattler/internal/battle"
	"github.com/DevJoshBrown/BeatBattler/internal/battle_participants"
	"github.com/DevJoshBrown/BeatBattler/internal/config"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/hub"
	"github.com/DevJoshBrown/BeatBattler/internal/matchmaker"
	"github.com/DevJoshBrown/BeatBattler/internal/middleware"
	"github.com/DevJoshBrown/BeatBattler/internal/queue"
	"github.com/DevJoshBrown/BeatBattler/internal/scheduler"
	"github.com/DevJoshBrown/BeatBattler/internal/user"
	votes "github.com/DevJoshBrown/BeatBattler/internal/vote"
	"github.com/DevJoshBrown/BeatBattler/pkg/storage/postgres"
	clerkSDK "github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	// Load config (internal/config)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	clerkSDK.SetKey(cfg.ClerkSecretKey)

	// Create pool (pkg/storage/postgres)
	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create a connection pool")
	}
	log.Printf("database connected - connections: %v", pool.Stat().TotalConns())
	defer pool.Close()

	//create queries & handlers (internal/db - sqlc generated)
	hubManager := hub.NewManager()
	queries := db.New(pool)
	userHandler := user.NewHandler(queries)
	sched := scheduler.NewScheduler(queries, pool, hubManager)
	battleHandler := battle.NewHandler(queries, sched, hubManager)
	participantHandler := battle_participants.NewHandler(queries)
	voteHandler := votes.NewHandler(queries)
	queueHandler := queue.NewHandler(queries)
	audioHandler := audio.NewHandler(queries)

	//matchmaker register
	mm := matchmaker.New(queries, sched)
	go mm.Run(context.Background())

	// Register routes on chi router
	r := chi.NewRouter()

	// Add CORS middleware

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-User-ID"},
	}))

	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})

	// Check server health
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.ClerkAuth)
		// Handlers
		// WebSocket
		r.Get("/battles/{id}/ws", battleHandler.ServeWS)
		// users
		r.Post("/users", userHandler.CreateUser)
		r.Post("/users/sync", userHandler.SyncUser)
		r.Get("/users/{id}", userHandler.GetUser)
		// battles
		r.Post("/battles", battleHandler.CreateBattle)
		r.Get("/battles", battleHandler.ListBattles)
		r.Get("/battles/{id}", battleHandler.GetBattle)
		r.Post("/battles/{id}/start", battleHandler.StartBattle)
		r.Get("/battles/{id}/results", battleHandler.GetResults)
		r.Get("/battles/{id}/audio/{participant_id}", audioHandler.GetTranscodedAudio)
		// matchmaking queue
		r.Post("/queue", queueHandler.Join)
		r.Delete("/queue", queueHandler.Leave)
		r.Get("/queue", queueHandler.Status)
		// participants
		r.Post("/battles/{id}/join", participantHandler.CreateParticipant)
		r.Post("/battles/{id}/submit", participantHandler.SubmitParticipant)
		r.Get("/battles/{id}/participants", participantHandler.ListParticipants)
		// votes
		r.Post("/battles/{id}/vote", voteHandler.CastVote)
		r.Post("/battles/{id}/confirm-votes", voteHandler.ConfirmVotes)
	})

	// Start server
	addr := ":" + cfg.Port
	log.Printf("BeatBattler listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
