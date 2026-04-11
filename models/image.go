package models

import (
	"database/sql"
	"database/sql/driver"
	"strconv"
)

type ItemImage struct {
	ID       int    `json:"id"`
	ItemID   int    `json:"item_id"`
	FilePath string `json:"file_path"`
}

func CreateItemImage(db *sql.DB, itemID int, filePath string) (ItemImage, error) {
	var img ItemImage
	err := db.QueryRow(
		"INSERT INTO item_images (item_id, file_path) VALUES ($1, $2) RETURNING id, item_id, file_path",
		itemID, filePath,
	).Scan(&img.ID, &img.ItemID, &img.FilePath)
	return img, err
}

func GetItemImage(db *sql.DB, id int) (ItemImage, error) {
	var img ItemImage
	err := db.QueryRow("SELECT id, item_id, file_path FROM item_images WHERE id = $1", id).Scan(&img.ID, &img.ItemID, &img.FilePath)
	return img, err
}

func ListItemImages(db *sql.DB, itemID int) ([]ItemImage, error) {
	rows, err := db.Query("SELECT id, item_id, file_path FROM item_images WHERE item_id = $1 ORDER BY id", itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imgs []ItemImage
	for rows.Next() {
		var img ItemImage
		if err := rows.Scan(&img.ID, &img.ItemID, &img.FilePath); err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	return imgs, rows.Err()
}

// ListItemImagesByItems returns images for multiple items, keyed by item ID.
func ListItemImagesByItems(db *sql.DB, itemIDs []int) (map[int][]ItemImage, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}

	rows, err := db.Query(
		"SELECT id, item_id, file_path FROM item_images WHERE item_id = ANY($1) ORDER BY item_id, id",
		intArray(itemIDs),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int][]ItemImage)
	for rows.Next() {
		var img ItemImage
		if err := rows.Scan(&img.ID, &img.ItemID, &img.FilePath); err != nil {
			return nil, err
		}
		m[img.ItemID] = append(m[img.ItemID], img)
	}
	return m, rows.Err()
}

func DeleteItemImage(db *sql.DB, imageID int) (string, error) {
	var filePath string
	err := db.QueryRow("DELETE FROM item_images WHERE id = $1 RETURNING file_path", imageID).Scan(&filePath)
	return filePath, err
}

func CountItemImages(db *sql.DB, itemID int) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM item_images WHERE item_id = $1", itemID).Scan(&count)
	return count, err
}

// intArray implements driver.Valuer for PostgreSQL int[] syntax.
type intArray []int

func (a intArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	s := "{"
	for i, v := range a {
		if i > 0 {
			s += ","
		}
		s += strconv.Itoa(v)
	}
	s += "}"
	return s, nil
}
