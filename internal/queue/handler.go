package queue

import (
	"encoding/json"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h Handler) Join(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	var usr_uid pgtype.UUID
	if err := usr_uid.Scan(userID); err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var body struct {
		Genres []string `json:"genres"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(body.Genres) == 0 {
		http.Error(w, "at least one genre required", http.StatusBadRequest)
		return
	}

	ticket, err := h.queries.EnqueueUser(r.Context(), db.EnqueueUserParams{
		UserID: usr_uid,
		Genres: body.Genres,
	})

	if err != nil {
		http.Error(w, "failed to join queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ticket)
}

func (h Handler) Leave(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	var usr_uid pgtype.UUID
	if err := usr_uid.Scan(userID); err != nil {
		http.Error(w, "invalid user_ID", http.StatusBadRequest)
		return
	}

	if err := h.queries.DequeueUser(r.Context(), usr_uid); err != nil {
		http.Error(w, "Failed to remove user from the queue", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) Status(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	var usr_uid pgtype.UUID
	if err := usr_uid.Scan(userID); err != nil {
		http.Error(w, "Failed to fetch userID", http.StatusBadRequest)
		return
	}

	ticket, err := h.queries.GetQueueTicket(r.Context(), usr_uid)
	if err != nil {
		http.Error(w, "not in queue", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)

}
