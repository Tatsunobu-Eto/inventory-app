package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type Item struct {
	ID           int        `json:"id"`
	DepartmentID int        `json:"department_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	OwnerID      int        `json:"owner_id"`
	CreatedBy    int        `json:"created_by"`
	Status       string     `json:"status"` // private, market, deleted
	MarketAt     *time.Time `json:"market_at"`
	CreatedAt    time.Time  `json:"created_at"`
	// joined fields
	OwnerName      string `json:"owner_name"`
	DepartmentName string `json:"department_name"`
}

type ItemFilter struct {
	DepartmentID int
	Status       string
	OwnerID      int
	Query        string
	Limit        int
	Offset       int
}

func ListItems(db *sql.DB, f ItemFilter) ([]Item, error) {
	q := `SELECT i.id, i.department_id, i.title, i.description, i.owner_id, i.created_by,
	             i.status, i.market_at, i.created_at, u.display_name, d.name
	      FROM items i
	      JOIN users u ON u.id = i.owner_id
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
			&it.OwnerID, &it.CreatedBy, &it.Status, &it.MarketAt, &it.CreatedAt,
			&it.OwnerName, &it.DepartmentName); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

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

func GetItem(db *sql.DB, id int) (Item, error) {
	var it Item
	err := db.QueryRow(
		`SELECT i.id, i.department_id, i.title, i.description, i.owner_id, i.created_by,
		        i.status, i.market_at, i.created_at, u.display_name
		 FROM items i JOIN users u ON u.id = i.owner_id
		 WHERE i.id = $1`, id,
	).Scan(&it.ID, &it.DepartmentID, &it.Title, &it.Description,
		&it.OwnerID, &it.CreatedBy, &it.Status, &it.MarketAt, &it.CreatedAt, &it.OwnerName)
	return it, err
}

func CreateItem(db *sql.DB, departmentID int, title, description string, ownerID, createdBy int) (Item, error) {
	var it Item
	err := db.QueryRow(
		`INSERT INTO items (department_id, title, description, owner_id, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, department_id, title, description, owner_id, created_by, status, market_at, created_at`,
		departmentID, title, description, ownerID, createdBy,
	).Scan(&it.ID, &it.DepartmentID, &it.Title, &it.Description,
		&it.OwnerID, &it.CreatedBy, &it.Status, &it.MarketAt, &it.CreatedAt)
	return it, err
}

func UpdateItemDescription(db *sql.DB, itemID int, description string) error {
	_, err := db.Exec("UPDATE items SET description = $1 WHERE id = $2", description, itemID)
	return err
}

func PutItemOnMarket(db *sql.DB, itemID, ownerID int) error {
	_, err := db.Exec(
		"UPDATE items SET status = 'market', market_at = NOW() WHERE id = $1 AND owner_id = $2 AND status = 'private'",
		itemID, ownerID,
	)
	return err
}

func WithdrawItem(db *sql.DB, itemID, ownerID int) error {
	_, err := db.Exec(
		"UPDATE items SET status = 'private', market_at = NULL WHERE id = $1 AND owner_id = $2 AND status = 'market'",
		itemID, ownerID,
	)
	return err
}

// ApplyForItem attempts to claim a market item with row-level locking.
// Returns (success, error).
func ApplyForItem(db *sql.DB, itemID, applicantID int) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var status string
	var ownerID int
	err = tx.QueryRow("SELECT status, owner_id FROM items WHERE id = $1 FOR UPDATE", itemID).Scan(&status, &ownerID)
	if err != nil {
		return false, err
	}

	if status != "market" {
		return false, nil
	}

	_, err = tx.Exec(
		"UPDATE items SET status = 'private', owner_id = $1, market_at = NULL WHERE id = $2",
		applicantID, itemID,
	)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (item_id, from_user_id, to_user_id) VALUES ($1, $2, $3)",
		itemID, ownerID, applicantID,
	)
	if err != nil {
		return false, err
	}

	return true, tx.Commit()
}

// ExpireMarketItems marks items that have been on market for over 90 days as deleted.
// Returns the IDs of all newly expired items so the caller can clean up their files.
func ExpireMarketItems(db *sql.DB) ([]int, error) {
	rows, err := db.Query(
		"UPDATE items SET status = 'deleted' WHERE status = 'market' AND market_at < NOW() - INTERVAL '90 days' RETURNING id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteItem removes an item record and returns the file paths of its images so the caller
// can delete them from disk. Returns an error if the item has transaction history.
func DeleteItem(db *sql.DB, itemID int) ([]string, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions WHERE item_id = $1", itemID).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("このアイテムには取引履歴があるため削除できません")
	}

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

func itoa(n int) string {
	return strconv.Itoa(n)
}
