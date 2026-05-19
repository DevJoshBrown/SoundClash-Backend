package battle

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h Handler) CreateBattle(w http.ResponseWriter, r *http.Request) {
	var params db.CreateBattleParams

	// Scan to UID
	id := r.Header.Get("X-User-ID")
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	params.CreatorID = uid

	b, err := h.queries.CreateBattle(r.Context(), params)
	if err != nil {
		log.Printf("CreateBattle error: %v", err)
		http.Error(w, "failed to create battle", http.StatusInternalServerError)
		return
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
	user_id := r.Header.Get("X-User-ID")
	var usr_uid pgtype.UUID
	if err := usr_uid.Scan(user_id); err != nil {
		http.Error(w, "could not scan user id", http.StatusBadRequest)
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

	if usr_uid != b.CreatorID {
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

	// assign positions (1-indexed)
	for i := range results {
		results[i].Position = i + 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
