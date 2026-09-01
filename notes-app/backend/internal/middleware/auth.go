// Package middleware holds HTTP middleware: JWT auth checking and CORS.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"notes-app/internal/auth"
)

// contextKey is a private type so other packages can't collide with our
// context keys by accident.
type contextKey string

const userIDKey contextKey = "userID"

// RequireAuth wraps a handler so it only runs if the request carries a
// valid "Authorization: Bearer <token>" header. On success, the user's
// ID is stored in the request context for handlers to read.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, http.StatusUnauthorized, "Authorization header must be: Bearer <token>")
				return
			}

			claims, err := auth.ValidateToken(parts[1], jwtSecret)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "your session has expired - please log in again")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext reads the authenticated user's ID that RequireAuth
// stored on the request context.
func UserIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
