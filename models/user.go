package models

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int    `json:"id"`
	DepartmentID *int   `json:"department_id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"` // sysadmin, admin, user
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GetUserByUsername(db *sql.DB, username string) (User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role)
	return u, err
}

func GetUserByID(db *sql.DB, id int) (User, error) {
	var u User
	err := db.QueryRow(
		"SELECT id, department_id, username, password_hash, display_name, role FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.DepartmentID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role)
	return u, err
}

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

func CountSysadmins(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'sysadmin'").Scan(&count)
	return count, err
}

func UpdateUserRole(db *sql.DB, userID int, role string) error {
	_, err := db.Exec("UPDATE users SET role = $1 WHERE id = $2", role, userID)
	return err
}

// UpdateUserDepartment はユーザーの所属部門を変更し、そのユーザーが owner_id を持つ
// 全アイテムの department_id も同じ部門に更新する。
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

func UpdatePassword(db *sql.DB, userID int, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hash, userID)
	return err
}

// DeleteUserCascade deletes a user along with their items and transactions.
// Returns the image file paths that should be removed from the filesystem.
func DeleteUserCascade(db *sql.DB, userID int) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Collect image paths for items owned or created by this user.
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

	// Delete transactions involving this user.
	if _, err := tx.Exec(
		"DELETE FROM transactions WHERE from_user_id = $1 OR to_user_id = $1", userID,
	); err != nil {
		return nil, err
	}

	// Delete items (item_images cascade automatically via ON DELETE CASCADE).
	if _, err := tx.Exec(
		"DELETE FROM items WHERE owner_id = $1 OR created_by = $1", userID,
	); err != nil {
		return nil, err
	}

	// Delete the user.
	if _, err := tx.Exec("DELETE FROM users WHERE id = $1", userID); err != nil {
		return nil, err
	}

	return paths, tx.Commit()
}
