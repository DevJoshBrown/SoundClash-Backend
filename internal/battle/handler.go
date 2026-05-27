package battle

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/DevJoshBrown/BeatBattler/internal/auth"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/hub"
	"github.com/DevJoshBrown/BeatBattler/internal/scheduler"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries   *db.Queries
	scheduler *scheduler.Scheduler
	hubs      *hub.Manager
}

func NewHandler(queries *db.Queries, s *scheduler.Scheduler, hubs *hub.Manager) *Handler {
	return &Handler{queries: queries, scheduler: s, hubs: hubs}
}

func (h Handler) CreateBattle(w http.ResponseWriter, r *http.Request) {
	var params db.CreateBattleParams

	// Scan to UID
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params.CreatorID = user.ID

	b, err := h.queries.CreateBattle(r.Context(), params)
	if err != nil {
		log.Printf("CreateBattle error: %v", err)
		http.Error(w, "failed to create battle", http.StatusInternalServerError)
		return
	}

	_, err = h.queries.CreateParticipant(r.Context(), db.CreateParticipantParams{
		BattleID: b.ID,
		UserID:   user.ID,
	})
	if err != nil {
		log.Printf("CreateBattle: failed to auto-join creator: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h Handler) GetBattle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(b)

}

func (h Handler) ListBattles(w http.ResponseWriter, r *http.Request) {
	b, err := h.queries.ListBattles(r.Context())
	if err != nil {
		http.Error(w, "battles not found", http.StatusInternalServerError)
		return
	}

	if b == nil {
		b = []db.Battle{}
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(b)
}

func (h Handler) StartBattle(w http.ResponseWriter, r *http.Request) {

	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID

	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if user.ID != b.CreatorID {
		http.Error(w, "user is not the battle creator", http.StatusForbidden)
		return
	}

	if b.Status != "waiting" {
		http.Error(w, "battle not currently waiting", http.StatusBadRequest)
		return
	}

	participants, err := h.queries.ListParticipants(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "could not list participants", http.StatusInternalServerError)
		return
	}

	if len(participants) < 2 {
		http.Error(w, "not enough participants to start", http.StatusBadRequest)
		return
	}

	battleStatus, err := h.queries.UpdateBattleStatus(r.Context(), db.UpdateBattleStatusParams{
		ID:     btl_uid,
		Status: "in_progress",
	})
	if err != nil {
		http.Error(w, "failed to update battle status", http.StatusInternalServerError)
		return
	}

	h.scheduler.Run(context.Background(), btl_uid, time.Duration(b.DurationMinutes)*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(battleStatus)

}

type ParticipantResult struct {
	Participant  db.BattleParticipant `json:"participant"`
	AverageScore float64              `json:"average_score"`
	Position     int                  `json:"position"`
	VoteCount    int                  `json:"vote_count"`
}

func (h Handler) GetResults(w http.ResponseWriter, r *http.Request) {
	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	if b.Status != "results" {
		http.Error(w, "results not available yet", http.StatusBadRequest)
		return
	}

	participants, err := h.queries.ListParticipants(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "failed to fetch participants", http.StatusInternalServerError)
		return
	}

	votes, err := h.queries.GetVotesForBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "failed to fetch votes", http.StatusInternalServerError)
		return
	}

	// sum scores and count votes per participant
	scoreSums := make(map[pgtype.UUID]int32)
	voteCounts := make(map[pgtype.UUID]int)
	for _, v := range votes {
		scoreSums[v.VotedForParticipantID] += v.Score
		voteCounts[v.VotedForParticipantID]++
	}

	// build results slice
	results := make([]ParticipantResult, 0, len(participants))
	for _, p := range participants {
		count := voteCounts[p.ID]
		avg := 0.0
		if count > 0 {
			avg = float64(scoreSums[p.ID]) / float64(count)
		}
		results = append(results, ParticipantResult{
			Participant:  p,
			AverageScore: avg,
			VoteCount:    count,
		})
	}

	// insertion sort by average score descending
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].AverageScore > results[j-1].AverageScore; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	// assign positions — tied scores share the same position
	for i := range results {
		if i == 0 {
			results[i].Position = 1
		} else if results[i].AverageScore == results[i-1].AverageScore {
			results[i].Position = results[i-1].Position
		} else {
			results[i].Position = i + 1
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(id); err != nil {
		http.Error(w, "invalid battle ID", http.StatusBadRequest)
		return
	}

	h.hubs.ServeWS(w, r, btl_uid)
}
