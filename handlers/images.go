package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"inventory-app/models"
)

const uploadDir = "uploads"
const maxUploadSize = 10 << 20 // 10MB

// SaveUploadedImages parses multipart form files and saves them to disk.
func (e *Env) SaveUploadedImages(r *http.Request, itemID int) error {
	if r.MultipartForm == nil {
		return nil
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		return nil
	}

	itemDir := filepath.Join(uploadDir, fmt.Sprintf("%d", itemID))
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return err
	}

	var savedFiles []string
	rollback := func() {
		for _, f := range savedFiles {
			if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
				log.Printf("warn: rollback failed to remove %s: %v", f, err)
			}
		}
	}

	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}

		src, err := fh.Open()
		if err != nil {
			rollback()
			return err
		}

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		// Always use forward slash for URL-safe DB storage
		relPath := fmt.Sprintf("%d/%s", itemID, filename)
		dstPath := filepath.Join(uploadDir, fmt.Sprintf("%d", itemID), filename)

		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			rollback()
			return err
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			os.Remove(dstPath)
			rollback()
			return err
		}

		savedFiles = append(savedFiles, dstPath)
		log.Printf("Uploaded image for item %d: %s\n", itemID, dstPath)

		if _, err := models.CreateItemImage(e.DB, itemID, relPath); err != nil {
			rollback()
			return err
		}
	}

	return nil
}
