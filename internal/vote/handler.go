package votes

import (
	"encoding/json"
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

func (h Handler) CastVote(w http.ResponseWriter, r *http.Request) {
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

	var body struct {
		ParticipantID string `json:"participant_id"`
		Score         int32  `json:"score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	var participant_uid pgtype.UUID

	if err := participant_uid.Scan(body.ParticipantID); err != nil {
		http.Error(w, "could not scan ParticipantId into a pgtype", http.StatusBadRequest)
		return
	}

	battle, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "failed to fetch battle", http.StatusNotFound)
		return
	}

	participant, err := h.queries.GetParticipant(r.Context(), db.GetParticipantParams{
		BattleID: btl_uid,
		UserID:   usr_uid,
	})

	if err != nil {
		http.Error(w, "failed to fetch participant", http.StatusNotFound)
		return
	}

	if battle.Status != "listening" && battle.Status != "voting" {
		http.Error(w, "voting not allowed at this stage", http.StatusBadRequest)
		return
	}

	if participant.UserID == usr_uid {
		http.Error(w, "self vote not allowed", http.StatusForbidden)
		return
	}

	target, err := h.queries.GetParticipantByID(r.Context(), participant_uid)
	if err != nil {
		http.Error(w, "no participant matches ID", http.StatusBadRequest)
		return
	}
	if target.BeatUrl.String == "" {
		http.Error(w, "participant is disqualified", http.StatusForbidden)
		return
	}

	vote, err := h.queries.UpsertVote(r.Context(), db.UpsertVoteParams{
		BattleID:              btl_uid,
		VoterID:               usr_uid,
		VotedForParticipantID: participant_uid,
		Score:                 body.Score,
	})
	if err != nil {
		http.Error(w, "failed to Upsert Vote", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vote)

}

func (h Handler) ConfirmVotes(w http.ResponseWriter, r *http.Request) {
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

	b, err := h.queries.GetBattle(r.Context(), btl_uid)
	if err != nil {
		http.Error(w, "battle not found", http.StatusNotFound)
		return
	}

	participant, err := h.queries.GetParticipant(r.Context(), db.GetParticipantParams{
		BattleID: btl_uid,
		UserID:   usr_uid,
	})
	if err != nil {
		http.Error(w, "participant not found", http.StatusBadRequest)
		return
	}

	if b.Status != "voting" {
		http.Error(w, "voting is not currently active", http.StatusBadRequest)
		return
	}

	participant, err = h.queries.ConfirmVotes(r.Context(), participant.ID)
	if err != nil {
		http.Error(w, "failed to confirm votes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participant)
}
