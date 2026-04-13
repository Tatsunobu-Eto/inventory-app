package models

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

// User はシステムのログインユーザーを表す。
// Role は "sysadmin"（システム管理者）/ "admin"（部門管理者）/ "user"（一般ユーザー）の3種類。
// DepartmentID は sysadmin の場合 nil になる。
type User struct {
	ID           int    `json:"id"`
	DepartmentID *int   `json:"department_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // JSON出力には含めない
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"` // sysadmin, admin, user
}

// HashPassword はパスワードをbcryptでハッシュ化して返す。
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword はハッシュとパスワードが一致するか検証する。一致すれば true を返す。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GetUserByUsername はユーザー名でユーザーを取得する。存在しない場合は sql.ErrNoRows を返す。
func GetUserByUsername(db *sql.DB, username string) (User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role)
	return u, err
}

// GetUserByID はユーザーIDでユーザーを取得する。
func GetUserByID(db *sql.DB, id int) (User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role)
	return u, err
}

// CreateUser はパスワードをハッシュ化してユーザーをDBに登録し、作成したユーザーを返す。
// departmentID は sysadmin 作成時に nil を渡す。
func CreateUser(db *sql.DB, departmentID *int, username, password, displayName, role string) (User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	var u User
	err = db.QueryRow(
		`INSERT INTO users (department_id, username, password_hash, display_name, role)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, department_id, username, password_hash, display_name, role`,
		departmentID, username, hash, displayName, role,
	).Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role)
	return u, err
}

// ListUsersByDepartment は指定部門に属するユーザーを表示名順で返す。
func ListUsersByDepartment(db *sql.DB, departmentID int) ([]User, error) {
	rows, err := db.Query(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE department_id = $1 ORDER BY display_name",
		departmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// ListAllUsers は部門に所属する全ユーザーを表示名順で返す（sysadmin は除外）。
func ListAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE department_id IS NOT NULL ORDER BY display_name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CountSysadmins はシステム管理者の人数を返す。初回起動時の初期ユーザー作成判定に使用する。
func CountSysadmins(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'sysadmin'").Scan(&count)
	return count, err
}

// UpdateUserRole はユーザーのロールを変更する（例: "user" → "admin"）。
func UpdateUserRole(db *sql.DB, userID int, role string) error {
	_, err := db.Exec("UPDATE users SET role = $1 WHERE id = $2", role, userID)
	return err
}

// UpdateUserDepartment はユーザーの所属部門を変更する。
// ユーザーが保有するアイテムの部門も同時に新しい部門へ更新し、データの整合性を保つ。
func UpdateUserDepartment(db *sql.DB, userID int, newDeptID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE users SET department_id = $1 WHERE id = $2", newDeptID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE items SET department_id = $1 WHERE owner_id = $2", newDeptID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdatePassword は新しいパスワードをハッシュ化してDBを更新する。
func UpdatePassword(db *sql.DB, userID int, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hash, userID)
	return err
}

// DeleteUserCascade はユーザーを削除し、関連するアイテム・取引履歴もまとめて削除する。
// ファイルシステム上の画像を削除できるよう、削除対象の画像パス一覧を返す。
func DeleteUserCascade(db *sql.DB, userID int) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 削除対象アイテムに紐づく画像パスを収集する（ファイル削除に使用）
	rows, err := tx.Query(
		`SELECT file_path FROM item_images
		 WHERE item_id IN (SELECT id FROM items WHERE owner_id = $1 OR created_by = $1)`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// このユーザーが送信者または受信者の取引履歴を削除する
	if _, err := tx.Exec(
		"DELETE FROM transactions WHERE from_user_id = $1 OR to_user_id = $1", userID,
	); err != nil {
		return nil, err
	}

	// アイテムを削除する（item_images は ON DELETE CASCADE で自動削除される）
	if _, err := tx.Exec(
		"DELETE FROM items WHERE owner_id = $1 OR created_by = $1", userID,
	); err != nil {
		return nil, err
	}

	// ユーザー本体を削除する
	if _, err := tx.Exec("DELETE FROM users WHERE id = $1", userID); err != nil {
		return nil, err
	}

	return paths, tx.Commit()
}
