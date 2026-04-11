package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"inventory-app/models"

	"github.com/gorilla/sessions"
)

type contextKey string

const UserKey contextKey = "user"

func CurrentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(UserKey).(*models.User)
	return u
}

// Auth loads the user from session and injects into context.
func Auth(db *sql.DB, store sessions.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := store.Get(r, "session")
			uid, ok := sess.Values["user_id"].(int)
			if !ok {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			user, err := models.GetUserByID(db, uid)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), UserKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that checks the user has one of the allowed roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := CurrentUser(r)
			if u == nil || !allowed[u.Role] {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
