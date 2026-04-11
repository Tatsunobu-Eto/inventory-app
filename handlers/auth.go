package handlers

import (
	"net/http"

	mw "inventory-app/middleware"
	"inventory-app/models"
)

func (e *Env) LoginPage(w http.ResponseWriter, r *http.Request) {
	e.render(w, "login.html", nil)
}

func (e *Env) LoginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// タイミング攻撃対策: ユーザーが存在しない場合もダミーハッシュでbcryptを実行し応答時間を均一化
	const dummyHash = "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	user, err := models.GetUserByUsername(e.DB, username)
	hashToCheck := dummyHash
	if err == nil {
		hashToCheck = user.PasswordHash
	}
	passwordOK := models.CheckPassword(hashToCheck, password)
	if err != nil || !passwordOK {
		triggerToast(w, "ユーザー名またはパスワードが正しくありません")
		e.render(w, "login.html", map[string]any{"Error": "ユーザー名またはパスワードが正しくありません"})
		return
	}

	sess, _ := e.Store.Get(r, "session")
	sess.Values["user_id"] = user.ID
	sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func (e *Env) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := e.Store.Get(r, "session")
	delete(sess.Values, "user_id")
	sess.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (e *Env) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	e.render(w, "dashboard.html", map[string]any{"User": user})
}

func (e *Env) PasswordChangePage(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	e.render(w, "password_change.html", map[string]any{"User": user})
}

func (e *Env) PasswordChangePost(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)

	current := r.FormValue("current_password")
	newPass := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if current == "" || newPass == "" || confirm == "" {
		triggerToast(w, "すべての項目を入力してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if newPass != confirm {
		triggerToast(w, "新しいパスワードと確認用パスワードが一致しません")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Re-fetch user to get current password hash
	dbUser, err := models.GetUserByID(e.DB, user.ID)
	if err != nil || !models.CheckPassword(dbUser.PasswordHash, current) {
		triggerToast(w, "現在のパスワードが正しくありません")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if err := models.UpdatePassword(e.DB, user.ID, newPass); err != nil {
		triggerToast(w, "パスワードの変更に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	triggerToast(w, "パスワードを変更しました")
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}
