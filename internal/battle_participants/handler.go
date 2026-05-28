package battle_participants

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DevJoshBrown/BeatBattler/internal/audio"
	"github.com/DevJoshBrown/BeatBattler/internal/auth"
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

	params.UserID = user.ID
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

	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
		UserID:   user.ID,
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

		//Set 32MB limit
		err = r.ParseMultipartForm(32 << 20)
		if err != nil {
			http.Error(w, "failed to parse multipart form", http.StatusInternalServerError)
			return
		}

		// get the audio from the form
		file, _, err := r.FormFile("audio")
		if err != nil {
			http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// save file to temp memory
		tmp, err := os.CreateTemp("", "beat-upload-*.tmp")
		if err != nil {
			http.Error(w, "failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmp.Name())

		io.Copy(tmp, file)
		tmp.Close()

		outputDir := fmt.Sprintf("tmp/%s/%s", battle_id, valid_participant.ID)
		outputPath, duration, err := audio.Transcode(tmp.Name(), outputDir)
		if err != nil {
			http.Error(w, "failed to transcode audio", http.StatusInternalServerError)
			return
		}

		_, err = h.queries.UpdateParticipantBeatURL(r.Context(), db.UpdateParticipantBeatURLParams{
			ID:          valid_participant.ID,
			BeatUrl:     pgtype.Text{String: outputPath, Valid: true},
			SubmittedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to update beat URL", http.StatusInternalServerError)
			return
		}

		_, err = h.queries.UpdateParticipantDuration(r.Context(), db.UpdateParticipantDurationParams{
			ID:              valid_participant.ID,
			DurationSeconds: pgtype.Int4{Int32: int32(duration), Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to update beat duration", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(valid_participant)

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
