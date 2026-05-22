package battle_participants

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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

func (h Handler) CreateParticipant(w http.ResponseWriter, r *http.Request) {
	var params db.CreateParticipantParams

	user_id := r.Header.Get("X-User-ID")
	var usr_uid pgtype.UUID

	if err := usr_uid.Scan(user_id); err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID

	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid battle id", http.StatusBadRequest)
		return
	}

	params.UserID = usr_uid
	params.BattleID = btl_uid

	p, err := h.queries.CreateParticipant(r.Context(), params)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, "failed to join battle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)

}

func (h Handler) SubmitParticipant(w http.ResponseWriter, r *http.Request) {

	// Get User
	user_id := r.Header.Get("X-User-ID")

	// convert id string to a pgtype uid
	var user_uid pgtype.UUID
	if err := user_uid.Scan(user_id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Get Battle
	battle_id := chi.URLParam(r, "id")

	var battle_uid pgtype.UUID
	if err := battle_uid.Scan(battle_id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	b, err := h.queries.GetBattle(r.Context(), battle_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	params := db.GetParticipantParams{
		BattleID: battle_uid,
		UserID:   user_uid,
	}

	// check user is a participant in the battle
	valid_participant, err := h.queries.GetParticipant(r.Context(), params)
	if err != nil {
		http.Error(w, "user is not a participant in this battle", http.StatusForbidden)
		return
	}

	// check the battle is still in progress
	if b.Status != "upload" {
		http.Error(w, "battle is not accepting uploads at this time", http.StatusBadRequest)
		return
	} else {
		var body struct {
			BeatURL string `json:"beat_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		updated, err := h.queries.UpdateParticipantBeatURL(r.Context(), db.UpdateParticipantBeatURLParams{
			ID:          valid_participant.ID,
			BeatUrl:     pgtype.Text{String: body.BeatURL, Valid: true},
			SubmittedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to updated participants beat URL", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)

	}
}

func (h Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	battle_id := chi.URLParam(r, "id")
	var btl_uid pgtype.UUID
	if err := btl_uid.Scan(battle_id); err != nil {
		http.Error(w, "failed to scan battle_id", http.StatusBadRequest)
		return
	}

	participants, err := h.queries.ListParticipants(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "failed to list participants", http.StatusInternalServerError)
		return
	}

	if len(participants) == 0 {
		participants = []db.BattleParticipant{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}
