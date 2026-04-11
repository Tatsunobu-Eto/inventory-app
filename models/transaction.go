package models

import (
	"database/sql"
	"time"
)

type Transaction struct {
	ID           int       `json:"id"`
	ItemID       int       `json:"item_id"`
	FromUserID   int       `json:"from_user_id"`
	ToUserID     int       `json:"to_user_id"`
	CreatedAt    time.Time `json:"created_at"`
	// joined fields
	ItemTitle    string `json:"item_title"`
	FromUserName string `json:"from_user_name"`
	ToUserName   string `json:"to_user_name"`
}

// CountTransactionsByUser returns the total number of transactions for the user.
func CountTransactionsByUser(db *sql.DB, userID int) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE from_user_id = $1 OR to_user_id = $1`,
		userID,
	).Scan(&count)
	return count, err
}

// ListTransactionsByUser returns transactions where the user is sender or receiver.
func ListTransactionsByUser(db *sql.DB, userID, limit, offset int) ([]Transaction, error) {
	rows, err := db.Query(`
		SELECT t.id, t.item_id, t.from_user_id, t.to_user_id, t.created_at,
		       i.title, fu.display_name, tu.display_name
		FROM transactions t
		JOIN items i ON i.id = t.item_id
		JOIN users fu ON fu.id = t.from_user_id
		JOIN users tu ON tu.id = t.to_user_id
		WHERE t.from_user_id = $1 OR t.to_user_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.ItemID, &tx.FromUserID, &tx.ToUserID, &tx.CreatedAt,
			&tx.ItemTitle, &tx.FromUserName, &tx.ToUserName); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}
