package auth

import (
	"errors"
	"net/http"

	"github.com/DevJoshBrown/BeatBattler/internal/db"
	"github.com/DevJoshBrown/BeatBattler/internal/middleware"
	"github.com/jackc/pgx/v5/pgtype"
)

func GetUserFromRequest(r *http.Request, queries *db.Queries) (db.User, error) {
	clerkID, ok := middleware.GetClerkUserID(r)
	if !ok {
		return db.User{}, errors.New("failed to fetch clerk ID")
	}

	pgClerk := pgtype.Text{String: clerkID, Valid: true}

	user, err := queries.GetUserByClerkID(r.Context(), pgClerk)
	if err != nil {
		return db.User{}, errors.New("failed to match user to clerkID")
	}
	return user, nil
}
