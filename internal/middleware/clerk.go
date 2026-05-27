package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2/jwt"
)

type contextKey string

const ClerkUserIDKey contextKey = "clerkUserID"

func ClerkAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
			Token: token,
		})
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ClerkUserIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClerkUserID(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(ClerkUserIDKey).(string)
	return id, ok
}
