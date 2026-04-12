package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"inventory-app/models"

	"github.com/gorilla/sessions"
)

// contextKey はリクエストコンテキストのキー型。文字列との混在を防ぐための専用型。
type contextKey string

// UserKey はコンテキストにログイン中ユーザーを格納する際に使うキー。
const UserKey contextKey = "user"

// CurrentUser はリクエストコンテキストからログイン中ユーザーを取り出す。
// Auth ミドルウェアが設定した値を返す。未ログイン時は nil。
func CurrentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(UserKey).(*models.User)
	return u
}

// Auth はセッションからユーザーIDを読み取り、DBでユーザー情報を確認してコンテキストに注入するミドルウェア。
// セッションが無効またはユーザーが存在しない場合は /login にリダイレクトする。
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

// RequireRole は指定されたロールを持つユーザーのみ通過を許可するミドルウェアを返す。
// ロールが一致しない場合は HTTP 403 Forbidden を返す。
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
