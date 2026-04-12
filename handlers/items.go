package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	mw "inventory-app/middleware"
	"inventory-app/models"

	"github.com/go-chi/chi/v5"
)

const maxImages = 5

// const uploadDir = "uploads" // 削除：handlers/images.goで定義済み

// collectImages はアイテム一覧に対応する画像をまとめて取得する。
// N+1クエリを避けるためアイテムIDをまとめて1回のDBクエリで取得する。
func collectImages(e *Env, items []models.Item) map[int][]models.ItemImage {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	m, _ := models.ListItemImagesByItems(e.DB, ids)
	if m == nil {
		m = make(map[int][]models.ItemImage)
	}
	return m
}

// MarketList はマーケット一覧を表示する。キーワード検索とページネーションに対応。
// HTMX からのリクエスト（HX-Request ヘッダあり）の場合はリスト部分のみを返す。
func (e *Env) MarketList(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)

	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	total, err := models.CountItems(e.DB, models.ItemFilter{
		Status: "market",
		Query:  query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	items, err := models.ListItems(e.DB, models.ItemFilter{
		Status: "market",
		Query:  query,
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imageMap := collectImages(e, items)

	data := map[string]any{
		"User":       user,
		"Items":      items,
		"ImageMap":   imageMap,
		"Query":      query,
		"Page":       page,
		"TotalPages": totalPages,
	}

	if r.Header.Get("HX-Request") == "true" {
		e.renderPartial(w, "item_list_partial.html", data)
		return
	}
	e.render(w, "market.html", data)
}

// MyItems はログインユーザーが所有するアイテム一覧を表示する。ページネーション対応。
func (e *Env) MyItems(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	total, err := models.CountItems(e.DB, models.ItemFilter{
		DepartmentID: *user.DepartmentID,
		OwnerID:      user.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	items, err := models.ListItems(e.DB, models.ItemFilter{
		DepartmentID: *user.DepartmentID,
		OwnerID:      user.ID,
		Limit:        perPage,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imageMap := collectImages(e, items)
	data := map[string]any{
		"User":       user,
		"Items":      items,
		"ImageMap":   imageMap,
		"Page":       page,
		"TotalPages": totalPages,
	}

	if r.Header.Get("HX-Request") == "true" {
		e.renderPartial(w, "my_items_partial.html", data)
		return
	}
	e.render(w, "my_items.html", data)
}

// CreateItemForm はアイテム登録フォームを表示する。
func (e *Env) CreateItemForm(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	e.render(w, "item_form.html", map[string]any{"User": user})
}

// CreateItemPost はアイテム登録フォームの送信を処理し、アイテムと画像をDBに保存する。
func (e *Env) CreateItemPost(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")

	if title == "" {
		triggerToast(w, "タイトルは必須です")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if r.MultipartForm != nil && len(r.MultipartForm.File["images"]) > maxImages {
		triggerToast(w, fmt.Sprintf("画像は最大%d枚までです", maxImages))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	item, err := models.CreateItem(e.DB, *user.DepartmentID, title, description, user.ID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := e.SaveUploadedImages(r, item.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "アイテムを登録しました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

// PutOnMarket はアイテムをマーケットに出品する。アイテムIDをフォームから受け取る。
func (e *Env) PutOnMarket(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, convErr := strconv.Atoi(r.FormValue("item_id"))
	if convErr != nil || itemID == 0 {
		http.Error(w, "無効なアイテムIDです", http.StatusBadRequest)
		return
	}

	if err := models.PutItemOnMarket(e.DB, itemID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "マーケットに出品しました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

// WithdrawFromMarket はマーケット出品を取り下げてアイテムを非公開に戻す。
func (e *Env) WithdrawFromMarket(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, convErr := strconv.Atoi(r.FormValue("item_id"))
	if convErr != nil || itemID == 0 {
		http.Error(w, "無効なアイテムIDです", http.StatusBadRequest)
		return
	}

	if err := models.WithdrawItem(e.DB, itemID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "出品を取り下げました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

// ApplyForItem はマーケット出品中のアイテムへの応募を処理する。
// 同じ部門のメンバーのみ応募可能。競合する同時応募はDBのトランザクションで制御される。
func (e *Env) ApplyForItem(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, convErr := strconv.Atoi(r.FormValue("item_id"))
	if convErr != nil || itemID == 0 {
		http.Error(w, "無効なアイテムIDです", http.StatusBadRequest)
		return
	}

	// 部門チェック: アイテムが自分の部門に属しているか確認
	item, err := models.GetItem(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user.DepartmentID == nil || item.DepartmentID != *user.DepartmentID {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}

	ok, err := models.ApplyForItem(e.DB, itemID, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusConflict)
		triggerToast(w, "既に他のユーザーが応募済みです")
		return
	}

	triggerToast(w, "応募が完了しました！")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

// ItemDetail はアイテムの詳細画面を表示する。
// クエリパラメータ "from" によって「戻る」リンクの遷移先を切り替える。
func (e *Env) ItemDetail(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	from := r.URL.Query().Get("from")
	if from == "" && user.Role == "sysadmin" {
		from = "sysadmin"
	}
	var backURL, backText string
	switch from {
	case "my-items":
		backURL, backText = "/my-items", "マイアイテムに戻る"
	case "admin":
		backURL, backText = "/admin/items", "部門アイテム一覧に戻る"
	case "sysadmin":
		backURL, backText = "/sysadmin/items", "全部門アイテム一覧に戻る"
	default:
		backURL, backText = "/market", "マーケットに戻る"
	}

	item, err := models.GetItem(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 部門チェック: sysadmin以外は自部門のアイテムのみアクセス可
	if user.Role != "sysadmin" && (user.DepartmentID == nil || item.DepartmentID != *user.DepartmentID) {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}

	// Check if the current user is the owner of the item
	isOwner := user.ID == item.OwnerID

	// Fetch images for the item
	images, err := models.ListItemImages(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":     user,
		"Item":     item,
		"Images":   images,
		"IsOwner":  isOwner,
		"BackURL":  backURL,
		"BackText": backText,
	}

	e.render(w, "item_detail.html", data)
}

// UpdateItemPost はアイテムの説明文と画像を更新する。アイテムの所有者のみ操作可能。
func (e *Env) UpdateItemPost(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := models.GetItem(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.ID != item.OwnerID {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}

	description := r.FormValue("description")
	// 現在は説明文のみ更新対象（タイトル変更が必要になった場合はここに追加）

	if err := models.UpdateItemDescription(e.DB, itemID, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 既存の画像枚数を確認し、追加後に上限を超えないかチェックする
	existingImages, err := models.ListItemImages(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currentImageCount := len(existingImages)

	if r.MultipartForm != nil && r.MultipartForm.File["images"] != nil {
		newFiles := r.MultipartForm.File["images"]
		if currentImageCount+len(newFiles) > maxImages {
			triggerToast(w, fmt.Sprintf("画像は最大%d枚までです。現在%d枚あり、%d枚追加しようとしています。", maxImages, currentImageCount, len(newFiles)))
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if err := e.SaveUploadedImages(r, item.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	triggerToast(w, "アイテムを更新しました")
	w.Header().Set("HX-Redirect", fmt.Sprintf("/items/%d", itemID))
	w.WriteHeader(http.StatusOK)
}

// Transactions はログインユーザーの取引履歴を表示する。ページネーション対応。
func (e *Env) Transactions(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	total, err := models.CountTransactionsByUser(e.DB, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	txs, err := models.ListTransactionsByUser(e.DB, user.ID, perPage, (page-1)*perPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":         user,
		"Transactions": txs,
		"Page":         page,
		"TotalPages":   totalPages,
	}

	if r.Header.Get("HX-Request") == "true" {
		e.renderPartial(w, "transactions_partial.html", data)
		return
	}
	e.render(w, "transactions.html", data)
}

// DeleteItem はアイテムをDBから削除し、紐づく画像ファイルもディスクから削除する。
// 所有者本人かつ非公開状態のアイテムのみ削除可能。取引履歴があるアイテムは削除不可。
func (e *Env) DeleteItem(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, err := strconv.Atoi(chi.URLParam(r, "item_id"))
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := models.GetItem(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.ID != item.OwnerID {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}

	if item.Status != "private" {
		triggerToast(w, "非公開状態のアイテムのみ削除できます")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	filePaths, err := models.DeleteItem(e.DB, itemID)
	if err != nil {
		triggerToast(w, err.Error())
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	for _, p := range filePaths {
		if err := os.Remove(filepath.Join(uploadDir, p)); err != nil && !os.IsNotExist(err) {
			log.Printf("warn: failed to remove file %s: %v", p, err)
		}
	}
	if err := os.Remove(filepath.Join(uploadDir, strconv.Itoa(itemID))); err != nil && !os.IsNotExist(err) {
		log.Printf("warn: failed to remove dir for item %d: %v", itemID, err)
	}

	triggerToast(w, "アイテムを削除しました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

// DeleteItemImage はアイテムの画像1枚をDBとディスクから削除する。アイテムの所有者のみ操作可能。
func (e *Env) DeleteItemImage(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	imageID, err := strconv.Atoi(chi.URLParam(r, "image_id"))
	if err != nil {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	image, err := models.GetItemImage(e.DB, imageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item, err := models.GetItem(e.DB, image.ItemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.ID != item.OwnerID {
		http.Error(w, "権限がありません", http.StatusForbidden)
		return
	}

	filePath, err := models.DeleteItemImage(e.DB, imageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.Remove(filepath.Join(uploadDir, filePath)); err != nil && !os.IsNotExist(err) {
		log.Printf("warn: failed to remove file %s: %v", filePath, err)
	}
	triggerToast(w, "画像を削除しました")
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}
