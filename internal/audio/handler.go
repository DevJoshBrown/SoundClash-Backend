package audio

import (
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

func (h Handler) GetTranscodedAudio(w http.ResponseWriter, r *http.Request) {
	participant_id := chi.URLParam(r, "participant_id")
	var par_uid pgtype.UUID

	err := par_uid.Scan(participant_id)
	if err != nil {
		http.Error(w, "failed to scan participant_id", http.StatusBadRequest)
		return
	}

	participant, err := h.queries.GetParticipantByID(r.Context(), par_uid)
	if err != nil {
		log.Printf("audio: GetParticipantByID failed for %v: %v", par_uid, err)
		http.Error(w, "failed to fetch participant from the db", http.StatusInternalServerError)
		return
	}
	log.Printf("audio: beat_url=%q valid=%v", participant.BeatUrl.String, participant.BeatUrl.Valid)

	if !participant.BeatUrl.Valid || participant.BeatUrl.String == "" {
		http.Error(w, "no audio submitted", http.StatusNotFound)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, participant.BeatUrl.String)

}
