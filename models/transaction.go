package models

import (
	"database/sql"
	"time"
)

// Transaction はアイテムの譲渡履歴1件を表す。
// FromUserID が元の所有者、ToUserID が新しい所有者（応募した側）。
// ItemTitle・FromUserName・ToUserName はSQLのJOINで取得した表示用フィールド。
// FromUserRead は元の所有者が取引履歴を確認したかどうかを示す。
type Transaction struct {
	ID           int       `json:"id"`
	ItemID       int       `json:"item_id"`
	FromUserID   int       `json:"from_user_id"`
	ToUserID     int       `json:"to_user_id"`
	CreatedAt    time.Time `json:"created_at"`
	FromUserRead bool      `json:"from_user_read"`
	// JOINで取得する表示用フィールド
	ItemTitle    string `json:"item_title"`
	FromUserName string `json:"from_user_name"`
	ToUserName   string `json:"to_user_name"`
}

// CountTransactionsByUser は指定ユーザーが関わる取引の総件数を返す（送信・受信どちらも含む）。
func CountTransactionsByUser(db *sql.DB, userID int) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE from_user_id = $1 OR to_user_id = $1`,
		userID,
	).Scan(&count)
	return count, err
}

// ListTransactionsByUser は指定ユーザーが送信者または受信者の取引履歴を新しい順で返す。
// ページネーション用に limit と offset を指定できる。
func ListTransactionsByUser(db *sql.DB, userID, limit, offset int) ([]Transaction, error) {
	rows, err := db.Query(`
		SELECT t.id, t.item_id, t.from_user_id, t.to_user_id, t.created_at, t.from_user_read,
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
		if err := rows.Scan(&tx.ID, &tx.ItemID, &tx.FromUserID, &tx.ToUserID, &tx.CreatedAt, &tx.FromUserRead,
			&tx.ItemTitle, &tx.FromUserName, &tx.ToUserName); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

// CountUnreadTransactionsForUser は指定ユーザーが元の所有者（出品者）として関わる未読取引の件数を返す。
func CountUnreadTransactionsForUser(db *sql.DB, userID int) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE from_user_id = $1 AND from_user_read = false`,
		userID,
	).Scan(&count)
	return count, err
}

// MarkTransactionsReadForUser は指定ユーザーが元の所有者（出品者）として関わる未読取引を全て既読にする。
func MarkTransactionsReadForUser(db *sql.DB, userID int) error {
	_, err := db.Exec(
		`UPDATE transactions SET from_user_read = true WHERE from_user_id = $1 AND from_user_read = false`,
		userID,
	)
	return err
}

// GetLatestTransactionForItem は指定アイテムの最新の取引を返す（応募成功ページの表示用）。
func GetLatestTransactionForItem(db *sql.DB, itemID int) (*Transaction, error) {
	var tx Transaction
	err := db.QueryRow(`
		SELECT t.id, t.item_id, t.from_user_id, t.to_user_id, t.created_at, t.from_user_read,
		       i.title, fu.display_name, tu.display_name
		FROM transactions t
		JOIN items i ON i.id = t.item_id
		JOIN users fu ON fu.id = t.from_user_id
		JOIN users tu ON tu.id = t.to_user_id
		WHERE t.item_id = $1
		ORDER BY t.created_at DESC
		LIMIT 1
	`, itemID).Scan(
		&tx.ID, &tx.ItemID, &tx.FromUserID, &tx.ToUserID, &tx.CreatedAt, &tx.FromUserRead,
		&tx.ItemTitle, &tx.FromUserName, &tx.ToUserName,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
