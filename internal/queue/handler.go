package queue

import (
	"encoding/json"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/auth"
	"github.com/DevJoshBrown/BeatBattler/internal/db"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h Handler) Join(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
		UserID: user.ID,
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
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.queries.DequeueUser(r.Context(), user.ID); err != nil {
		http.Error(w, "Failed to remove user from the queue", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) Status(w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserFromRequest(r, h.queries)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ticket, err := h.queries.GetQueueTicket(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "not in queue", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)

}
