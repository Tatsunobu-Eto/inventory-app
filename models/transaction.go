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
	ItemTitle        string `json:"item_title"`
	ItemStatus       string `json:"item_status"`
	FromUserName     string `json:"from_user_name"`
	FromUserDeptName string `json:"from_user_dept_name"`
	ToUserName       string `json:"to_user_name"`
	ToUserDeptName   string `json:"to_user_dept_name"`
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
		       i.title, i.status, fu.display_name, COALESCE(fd.name, ''), tu.display_name, COALESCE(td.name, '')
		FROM transactions t
		JOIN items i ON i.id = t.item_id
		JOIN users fu ON fu.id = t.from_user_id
		LEFT JOIN departments fd ON fd.id = fu.department_id
		JOIN users tu ON tu.id = t.to_user_id
		LEFT JOIN departments td ON td.id = tu.department_id
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
			&tx.ItemTitle, &tx.ItemStatus, &tx.FromUserName, &tx.FromUserDeptName, &tx.ToUserName, &tx.ToUserDeptName); err != nil {
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

// ApproveApplication は応募を承認する。
// 所有権を応募者に移し、アイテムを "private" 状態にした上で応募レコードを削除する。
func ApproveApplication(db *sql.DB, txID, fromUserID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var itemID, toUserID int
	err = tx.QueryRow(
		"SELECT item_id, to_user_id FROM transactions WHERE id = $1 AND from_user_id = $2 FOR UPDATE",
		txID, fromUserID,
	).Scan(&itemID, &toUserID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"UPDATE items SET status = 'private', owner_id = $1 WHERE id = $2",
		toUserID, itemID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM transactions WHERE id = $1", txID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RejectApplication は応募を拒否する。
// アイテムをマーケットに戻し、応募レコードを削除する。
func RejectApplication(db *sql.DB, txID, fromUserID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var itemID int
	err = tx.QueryRow(
		"SELECT item_id FROM transactions WHERE id = $1 AND from_user_id = $2 FOR UPDATE",
		txID, fromUserID,
	).Scan(&itemID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"UPDATE items SET status = 'market' WHERE id = $1",
		itemID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM transactions WHERE id = $1", txID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetLatestTransactionForItem は指定アイテムの最新の取引を返す（応募成功ページの表示用）。
func GetLatestTransactionForItem(db *sql.DB, itemID int) (*Transaction, error) {
	var tx Transaction
	err := db.QueryRow(`
		SELECT t.id, t.item_id, t.from_user_id, t.to_user_id, t.created_at, t.from_user_read,
		       i.title, i.status, fu.display_name, COALESCE(fd.name, ''), tu.display_name, COALESCE(td.name, '')
		FROM transactions t
		JOIN items i ON i.id = t.item_id
		JOIN users fu ON fu.id = t.from_user_id
		LEFT JOIN departments fd ON fd.id = fu.department_id
		JOIN users tu ON tu.id = t.to_user_id
		LEFT JOIN departments td ON td.id = tu.department_id
		WHERE t.item_id = $1
		ORDER BY t.created_at DESC
		LIMIT 1
	`, itemID).Scan(
		&tx.ID, &tx.ItemID, &tx.FromUserID, &tx.ToUserID, &tx.CreatedAt, &tx.FromUserRead,
		&tx.ItemTitle, &tx.ItemStatus, &tx.FromUserName, &tx.FromUserDeptName, &tx.ToUserName, &tx.ToUserDeptName,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
