package models

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

// ErrSelfApplication は自分が出品したアイテムに応募しようとした場合のエラー。
var ErrSelfApplication = errors.New("自分のアイテムには応募できません")

// Item は消耗品1件を表す。
// Status は "private"（非公開）/ "market"（マーケット出品中）/ "applying"（承認待ち）の3種類。
// OwnerName・DepartmentName はSQLのJOINで取得した表示用フィールド。
type Item struct {
	ID           int       `json:"id"`
	DepartmentID int       `json:"department_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	OwnerID      int       `json:"owner_id"`
	CreatedBy    int       `json:"created_by"`
	Status       string    `json:"status"` // private, market, applying
	CreatedAt    time.Time `json:"created_at"`
	// JOINで取得する表示用フィールド
	OwnerName      string `json:"owner_name"`
	OwnerDeptName  string `json:"owner_dept_name"`
	DepartmentName string `json:"department_name"`
}

// ItemFilter はアイテム一覧取得時の絞り込み条件をまとめた構造体。
// 0値（ゼロ値）のフィールドはフィルタとして無視される。
type ItemFilter struct {
	DepartmentID int    // 部門で絞り込む（0なら全部門）
	Status       string // ステータスで絞り込む（空文字なら全ステータス）
	OwnerID      int    // 所有者で絞り込む（0なら全ユーザー）
	Query        string // タイトル・説明文のキーワード検索
	Limit        int    // 取得件数（0なら制限なし）
	Offset       int    // 取得開始位置（ページネーション用）
}

// ListItems はフィルタ条件に合うアイテム一覧を作成日時の降順で返す。
// WHERE句をフィルタの内容に応じて動的に組み立てる。
func ListItems(db *sql.DB, f ItemFilter) ([]Item, error) {
	q := `SELECT i.id, i.department_id, i.title, i.description, i.owner_id, i.created_by,
	             i.status, i.created_at, u.display_name, COALESCE(du.name, ''), d.name
	      FROM items i
	      JOIN users u ON u.id = i.owner_id
	      LEFT JOIN departments du ON du.id = u.department_id
	      JOIN departments d ON d.id = i.department_id`
	args := []any{}
	n := 1

	if f.DepartmentID != 0 {
		q += " WHERE i.department_id = $" + itoa(n)
		args = append(args, f.DepartmentID)
		n++
	}

	addWhere := func(cond string, val any) {
		if len(args) == 0 {
			q += " WHERE " + cond
		} else {
			q += " AND " + cond
		}
		args = append(args, val)
		n++
	}

	if f.Status != "" {
		addWhere("i.status = $"+itoa(n), f.Status)
	}
	if f.OwnerID != 0 {
		addWhere("i.owner_id = $"+itoa(n), f.OwnerID)
	}
	if f.Query != "" {
		if len(args) == 0 {
			q += " WHERE (i.title ILIKE $" + itoa(n) + " OR i.description ILIKE $" + itoa(n) + ")"
		} else {
			q += " AND (i.title ILIKE $" + itoa(n) + " OR i.description ILIKE $" + itoa(n) + ")"
		}
		args = append(args, "%"+f.Query+"%")
		n++
	}

	q += " ORDER BY i.created_at DESC"

	if f.Limit > 0 {
		q += " LIMIT $" + itoa(n)
		args = append(args, f.Limit)
		n++
	}
	if f.Offset > 0 {
		q += " OFFSET $" + itoa(n)
		args = append(args, f.Offset)
		n++
	}

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.DepartmentID, &it.Title, &it.Description,
			&it.OwnerID, &it.CreatedBy, &it.Status, &it.CreatedAt,
			&it.OwnerName, &it.OwnerDeptName, &it.DepartmentName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// CountItems はフィルタ条件に合うアイテムの総件数を返す。ページネーションの総ページ数計算に使用する。
func CountItems(db *sql.DB, f ItemFilter) (int, error) {
	q := `SELECT COUNT(*) FROM items i`
	args := []any{}
	n := 1

	if f.DepartmentID != 0 {
		q += " WHERE i.department_id = $" + itoa(n)
		args = append(args, f.DepartmentID)
		n++
	}

	addWhere := func(cond string, val any) {
		if len(args) == 0 {
			q += " WHERE " + cond
		} else {
			q += " AND " + cond
		}
		args = append(args, val)
		n++
	}

	if f.Status != "" {
		addWhere("i.status = $"+itoa(n), f.Status)
	}
	if f.OwnerID != 0 {
		addWhere("i.owner_id = $"+itoa(n), f.OwnerID)
	}
	if f.Query != "" {
		if len(args) == 0 {
			q += " WHERE (i.title ILIKE $" + itoa(n) + " OR i.description ILIKE $" + itoa(n) + ")"
		} else {
			q += " AND (i.title ILIKE $" + itoa(n) + " OR i.description ILIKE $" + itoa(n) + ")"
		}
		args = append(args, "%"+f.Query+"%")
		n++
	}

	var count int
	err := db.QueryRow(q, args...).Scan(&count)
	return count, err
}

// GetItem は指定IDのアイテムを取得する。所有者の表示名もJOINで取得する。
func GetItem(db *sql.DB, id int) (Item, error) {
	var it Item
	err := db.QueryRow(
		`SELECT i.id, i.department_id, i.title, i.description, i.owner_id, i.created_by,
		        i.status, i.created_at, u.display_name
		 FROM items i JOIN users u ON u.id = i.owner_id
		 WHERE i.id = $1`, id,
	).Scan(&it.ID, &it.DepartmentID, &it.Title, &it.Description,
		&it.OwnerID, &it.CreatedBy, &it.Status, &it.CreatedAt, &it.OwnerName)
	return it, err
}

// CreateItem は新しいアイテムをDBに登録して返す。初期ステータスは "private"（非公開）。
// ownerID は現在の所有者、createdBy は登録操作を行ったユーザー（代理登録時に異なる）。
func CreateItem(db *sql.DB, departmentID int, title, description string, ownerID, createdBy int) (Item, error) {
	var it Item
	err := db.QueryRow(
		`INSERT INTO items (department_id, title, description, owner_id, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, department_id, title, description, owner_id, created_by, status, created_at`,
		departmentID, title, description, ownerID, createdBy,
	).Scan(&it.ID, &it.DepartmentID, &it.Title, &it.Description,
		&it.OwnerID, &it.CreatedBy, &it.Status, &it.CreatedAt)
	return it, err
}

// UpdateItemDescription はアイテムの説明文を更新する。
func UpdateItemDescription(db *sql.DB, itemID int, description string) error {
	_, err := db.Exec("UPDATE items SET description = $1 WHERE id = $2", description, itemID)
	return err
}

// PutItemOnMarket はアイテムをマーケットに出品する（status を "market" に変更）。
// 所有者本人かつ現在 "private" のアイテムのみ出品できる。
func PutItemOnMarket(db *sql.DB, itemID, ownerID int) error {
	_, err := db.Exec(
		"UPDATE items SET status = 'market' WHERE id = $1 AND owner_id = $2 AND status = 'private'",
		itemID, ownerID,
	)
	return err
}

// WithdrawItem はマーケット出品を取り下げてアイテムを非公開に戻す（status を "private" に変更）。
func WithdrawItem(db *sql.DB, itemID, ownerID int) error {
	_, err := db.Exec(
		"UPDATE items SET status = 'private' WHERE id = $1 AND owner_id = $2 AND status = 'market'",
		itemID, ownerID,
	)
	return err
}

// ApplyForItem はマーケット出品中のアイテムへの応募を処理する。
// 行レベルロック（FOR UPDATE）を使い、複数ユーザーの同時応募による二重譲渡を防ぐ。
// 成功した場合はアイテムが "applying"（承認待ち）状態になり、応募レコードが記録される。
// 所有権の移転は出品者が承認した時点で行われる（ApproveApplication を参照）。
// 戻り値: (成功したか, エラー)
func ApplyForItem(db *sql.DB, itemID, applicantID int) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var status string
	var ownerID int
	// FOR UPDATE でこの行をロックし、他のトランザクションが同時に処理するのを防ぐ
	err = tx.QueryRow("SELECT status, owner_id FROM items WHERE id = $1 FOR UPDATE", itemID).Scan(&status, &ownerID)
	if err != nil {
		return false, err
	}

	if applicantID == ownerID {
		return false, ErrSelfApplication
	}

	if status != "market" {
		return false, nil
	}

	// 所有権は移さず、承認待ち状態にする
	_, err = tx.Exec(
		"UPDATE items SET status = 'applying' WHERE id = $1",
		itemID,
	)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (item_id, from_user_id, to_user_id, from_user_read) VALUES ($1, $2, $3, false)",
		itemID, ownerID, applicantID,
	)
	if err != nil {
		return false, err
	}

	return true, tx.Commit()
}

// DeleteItem はアイテムをDBから削除し、紐づく画像のファイルパス一覧を返す。
func DeleteItem(db *sql.DB, itemID int) ([]string, error) {
	rows, err := db.Query("SELECT file_path FROM item_images WHERE item_id = $1", itemID)
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

	_, err = db.Exec("DELETE FROM items WHERE id = $1", itemID)
	return paths, err
}

// itoa は整数を文字列に変換するヘルパー。SQLのプレースホルダ番号生成に使用する。
func itoa(n int) string {
	return strconv.Itoa(n)
}
