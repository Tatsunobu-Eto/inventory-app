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

// uploadDir は画像ファイルの保存先ディレクトリ。
const uploadDir = "uploads"

// maxUploadSize はアップロード可能な1ファイルの最大サイズ（10MB）。
const maxUploadSize = 10 << 20 // 10MB

// SaveUploadedImages はマルチパートフォームから画像ファイルを受け取り、ディスクに保存してDBにも登録する。
// 対応形式: jpg/jpeg/png/gif/webp。途中でエラーが発生した場合は保存済みファイルを削除してロールバックする。
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
	// rollback はエラー発生時に保存済みファイルをすべて削除するクリーンアップ関数
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

		// ファイル名はナノ秒タイムスタンプで一意性を確保する
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		// DBには "/" 区切りのURLセーフなパスで保存する
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
