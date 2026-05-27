package user

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var params db.CreateUserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	u, err := h.queries.CreateUser(r.Context(), params)
	if err != nil {
		log.Printf("createUser error: %v", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (h Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// convert id string to a pgtype uid
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	u, err := h.queries.GetUserByID(r.Context(), uid)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (h Handler) SyncUser(w http.ResponseWriter, r *http.Request) {
	clerkID, ok := middleware.GetClerkUserID(r)
	if !ok {
		http.Error(w, "failed to fetch ClerkID", http.StatusUnauthorized)
		return
	}

	pgClerkID := pgtype.Text{String: clerkID, Valid: true}

	type SyncUserRequest struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}

	var req SyncUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}

	user, err := h.queries.UpsertUserByClerkID(r.Context(), db.UpsertUserByClerkIDParams{
		Username:    req.Username,
		DisplayName: req.DisplayName,
		ClerkID:     pgClerkID,
	})
	if err != nil {
		http.Error(w, "failed to Upset user from request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

}
