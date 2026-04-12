package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	mw "inventory-app/middleware"
	"inventory-app/models"

	"github.com/lib/pq"
)

// deptWithUsers は部門情報とその所属ユーザー一覧をまとめた表示用の構造体。
type deptWithUsers struct {
	models.Department
	Users []models.User
}

// --- システム管理者（sysadmin）用ハンドラ ---

// SysAdminDepartments は全部門とその所属ユーザー一覧を表示する。
func (e *Env) SysAdminDepartments(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	deps, err := models.ListDepartments(e.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	depts := make([]deptWithUsers, 0, len(deps))
	for _, d := range deps {
		users, err := models.ListUsersByDepartment(e.DB, d.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		depts = append(depts, deptWithUsers{Department: d, Users: users})
	}
	e.render(w, r, "sysadmin_departments.html", map[string]any{"User": user, "Depts": depts})
}

// SysAdminCreateDepartment は新しい部門を作成する。部門名が既に存在する場合はエラーを返す。
func (e *Env) SysAdminCreateDepartment(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		triggerToast(w, "部門名は必須です")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	_, err := models.CreateDepartment(e.DB, name)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			triggerToast(w, "その部門名はすでに存在します")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		triggerToast(w, "部門の作成に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	triggerToast(w, "部門を作成しました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminCreateAdmin は指定部門の部門管理者（admin）アカウントを作成する。
func (e *Env) SysAdminCreateAdmin(w http.ResponseWriter, r *http.Request) {
	deptID, convErr := strconv.Atoi(r.FormValue("department_id"))
	username := r.FormValue("username")
	password := r.FormValue("password")
	displayName := r.FormValue("display_name")

	if convErr != nil || username == "" || password == "" || deptID == 0 {
		triggerToast(w, "すべての項目を入力してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	_, err := models.CreateUser(e.DB, &deptID, username, password, displayName, "admin")
	if err != nil {
		triggerToast(w, "ユーザー作成に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	triggerToast(w, "部門管理者を作成しました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminPromoteToAdmin は一般ユーザーを部門管理者に昇格させる。
func (e *Env) SysAdminPromoteToAdmin(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, userID)
	if err != nil {
		triggerToast(w, "ユーザーが見つかりません")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if target.Role != "user" {
		triggerToast(w, "このユーザーは昇格できません")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdateUserRole(e.DB, userID, "admin"); err != nil {
		triggerToast(w, "ロールの更新に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+" を管理者にしました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminDemoteToUser は部門管理者を一般ユーザーに降格させる。
func (e *Env) SysAdminDemoteToUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, userID)
	if err != nil {
		triggerToast(w, "ユーザーが見つかりません")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if target.Role != "admin" {
		triggerToast(w, "このユーザーは降格できません")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdateUserRole(e.DB, userID, "user"); err != nil {
		triggerToast(w, "ロールの更新に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+" を一般ユーザーにしました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminAllItems は全部門のアイテム一覧を表示する。ステータス・ユーザー・キーワードで絞り込み可能。
func (e *Env) SysAdminAllItems(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)

	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	ownerID, _ := strconv.Atoi(r.URL.Query().Get("owner_id"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	total, err := models.CountItems(e.DB, models.ItemFilter{
		DepartmentID: 0,
		Status:       status,
		OwnerID:      ownerID,
		Query:        query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	items, err := models.ListItems(e.DB, models.ItemFilter{
		DepartmentID: 0, // 全部門
		Status:       status,
		OwnerID:      ownerID,
		Query:        query,
		Limit:        perPage,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	allUsers, err := models.ListAllUsers(e.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	imageMap := collectImages(e, items)

	e.render(w, r, "sysadmin_all_items.html", map[string]any{
		"User":       user,
		"Items":      items,
		"ImageMap":   imageMap,
		"Query":      query,
		"Status":     status,
		"OwnerID":    ownerID,
		"AllUsers":   allUsers,
		"Page":       page,
		"TotalPages": totalPages,
	})
}

// --- 部門管理者（admin）用ハンドラ ---

// AdminUsers は自分の部門のユーザー一覧と部門情報を表示する。
func (e *Env) AdminUsers(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	users, err := models.ListUsersByDepartment(e.DB, *user.DepartmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dept, _ := models.GetDepartment(e.DB, *user.DepartmentID)
	allDepts, err := models.ListDepartments(e.DB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	e.render(w, r, "admin_users.html", map[string]any{
		"User":        user,
		"Users":       users,
		"Department":  dept,
		"Departments": allDepts,
	})
}

// AdminCreateUser は自分の部門に一般ユーザーを新規作成する。
func (e *Env) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	currentUser := mw.CurrentUser(r)
	if currentUser.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	displayName := r.FormValue("display_name")

	if username == "" || password == "" {
		triggerToast(w, "すべての項目を入力してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	_, err := models.CreateUser(e.DB, currentUser.DepartmentID, username, password, displayName, "user")
	if err != nil {
		triggerToast(w, "ユーザー作成に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	triggerToast(w, "ユーザーを作成しました")
	w.Header().Set("HX-Redirect", "/admin/users")
	w.WriteHeader(http.StatusOK)
}

// AdminCreateItem は部門管理者が部門メンバーの代わりにアイテムを登録する（代理登録）。
func (e *Env) AdminCreateItem(w http.ResponseWriter, r *http.Request) {
	currentUser := mw.CurrentUser(r)
	if currentUser.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	ownerID, convErr := strconv.Atoi(r.FormValue("owner_id"))

	if convErr != nil || ownerID == 0 || title == "" {
		triggerToast(w, "タイトルと所有者は必須です")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// 部門チェック: ownerが同じ部門に属しているか確認
	owner, err := models.GetUserByID(e.DB, ownerID)
	if err != nil || owner.DepartmentID == nil || *owner.DepartmentID != *currentUser.DepartmentID {
		triggerToast(w, "同じ部門のユーザーのみを選択できます")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.MultipartForm != nil && len(r.MultipartForm.File["images"]) > maxImages {
		triggerToast(w, fmt.Sprintf("画像は最大%d枚までです", maxImages))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	item, err := models.CreateItem(e.DB, *currentUser.DepartmentID, title, description, ownerID, currentUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := e.SaveUploadedImages(r, item.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "消耗品を登録しました")
	w.Header().Set("HX-Redirect", "/admin/items/new")
	w.WriteHeader(http.StatusOK)
}

// AdminCreateItemForm はアイテム代理登録フォームを表示する。部門内ユーザー一覧も渡す。
func (e *Env) AdminCreateItemForm(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	users, err := models.ListUsersByDepartment(e.DB, *user.DepartmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	e.render(w, r, "admin_item_form.html", map[string]any{"User": user, "Users": users})
}

// AdminDeptItems は自分の部門のアイテム一覧を表示する。ステータス・ユーザー・キーワードで絞り込み可能。
func (e *Env) AdminDeptItems(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	ownerID, _ := strconv.Atoi(r.URL.Query().Get("owner_id"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	total, err := models.CountItems(e.DB, models.ItemFilter{
		DepartmentID: *user.DepartmentID,
		Query:        query,
		Status:       status,
		OwnerID:      ownerID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	items, err := models.ListItems(e.DB, models.ItemFilter{
		DepartmentID: *user.DepartmentID,
		Query:        query,
		Status:       status,
		OwnerID:      ownerID,
		Limit:        perPage,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dept, _ := models.GetDepartment(e.DB, *user.DepartmentID)
	deptUsers, err := models.ListUsersByDepartment(e.DB, *user.DepartmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	imageMap := collectImages(e, items)

	e.render(w, r, "admin_dept_items.html", map[string]any{
		"User":       user,
		"Items":      items,
		"ImageMap":   imageMap,
		"Department": dept,
		"Query":      query,
		"Status":     status,
		"OwnerID":    ownerID,
		"DeptUsers":  deptUsers,
		"Page":       page,
		"TotalPages": totalPages,
	})
}

// --- 部門管理者：ユーザー管理操作 ---

// AdminDeleteUser は自部門の一般ユーザーを削除する。管理者ロールのユーザーは削除不可。
func (e *Env) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := mw.CurrentUser(r)
	if currentUser.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}
	if target.DepartmentID == nil || *target.DepartmentID != *currentUser.DepartmentID || target.Role != "user" {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}
	paths, err := models.DeleteUserCascade(e.DB, targetID)
	if err != nil {
		triggerToast(w, "削除に失敗しました: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for _, p := range paths {
		if err := os.Remove(filepath.Join(uploadDir, p)); err != nil && !os.IsNotExist(err) {
			log.Printf("warn: failed to remove file %s: %v", p, err)
		}
	}
	triggerToast(w, target.DisplayName+"を削除しました")
	w.Header().Set("HX-Redirect", "/admin/users")
	w.WriteHeader(http.StatusOK)
}

// AdminResetPassword は自部門の一般ユーザーのパスワードを管理者がリセットする。
func (e *Env) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	currentUser := mw.CurrentUser(r)
	if currentUser.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}
	if target.DepartmentID == nil || *target.DepartmentID != *currentUser.DepartmentID || target.Role != "user" {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}
	newPassword := r.FormValue("new_password")
	if newPassword == "" {
		triggerToast(w, "新しいパスワードを入力してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdatePassword(e.DB, targetID, newPassword); err != nil {
		triggerToast(w, "パスワードリセットに失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+"のパスワードをリセットしました")
	w.Header().Set("HX-Redirect", "/admin/users")
	w.WriteHeader(http.StatusOK)
}

// AdminTransferUser は自部門の一般ユーザーを別の部門に異動させる。所有アイテムの部門も同時に変更される。
func (e *Env) AdminTransferUser(w http.ResponseWriter, r *http.Request) {
	currentUser := mw.CurrentUser(r)
	if currentUser.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		triggerToast(w, "ユーザーが見つかりません")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if target.DepartmentID == nil || *target.DepartmentID != *currentUser.DepartmentID || target.Role != "user" {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}
	newDeptID, err := strconv.Atoi(r.FormValue("department_id"))
	if err != nil || newDeptID == 0 {
		triggerToast(w, "異動先部門を選択してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if newDeptID == *currentUser.DepartmentID {
		triggerToast(w, "異動先は現在の部門と異なる部門を選択してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdateUserDepartment(e.DB, targetID, newDeptID); err != nil {
		triggerToast(w, "異動に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+" を異動しました")
	w.Header().Set("HX-Redirect", "/admin/users")
	w.WriteHeader(http.StatusOK)
}

// --- システム管理者：ユーザー管理操作 ---

// SysAdminDeleteUser は全ユーザーを対象にユーザーを削除する（全部門の範囲）。
func (e *Env) SysAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}
	paths, err := models.DeleteUserCascade(e.DB, targetID)
	if err != nil {
		triggerToast(w, "削除に失敗しました: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for _, p := range paths {
		if err := os.Remove(filepath.Join(uploadDir, p)); err != nil && !os.IsNotExist(err) {
			log.Printf("warn: failed to remove file %s: %v", p, err)
		}
	}
	triggerToast(w, target.DisplayName+"を削除しました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminResetPassword は全ユーザーを対象にパスワードをリセットする（全部門の範囲）。
func (e *Env) SysAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		http.Error(w, "ユーザーが見つかりません", http.StatusNotFound)
		return
	}
	newPassword := r.FormValue("new_password")
	if newPassword == "" {
		triggerToast(w, "新しいパスワードを入力してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdatePassword(e.DB, targetID, newPassword); err != nil {
		triggerToast(w, "パスワードリセットに失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+"のパスワードをリセットしました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}

// SysAdminTransferUser は全ユーザーを対象に部門異動を行う（全部門の範囲）。sysadmin 自身は異動不可。
func (e *Env) SysAdminTransferUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	target, err := models.GetUserByID(e.DB, targetID)
	if err != nil {
		triggerToast(w, "ユーザーが見つかりません")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if target.Role == "sysadmin" {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}
	newDeptID, err := strconv.Atoi(r.FormValue("department_id"))
	if err != nil || newDeptID == 0 {
		triggerToast(w, "異動先部門を選択してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if target.DepartmentID != nil && *target.DepartmentID == newDeptID {
		triggerToast(w, "異動先は現在の部門と異なる部門を選択してください")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if err := models.UpdateUserDepartment(e.DB, targetID, newDeptID); err != nil {
		triggerToast(w, "異動に失敗しました")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	triggerToast(w, target.DisplayName+" を異動しました")
	w.Header().Set("HX-Redirect", "/sysadmin/departments")
	w.WriteHeader(http.StatusOK)
}
