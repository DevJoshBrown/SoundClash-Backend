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
