package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	mw "inventory-app/middleware"
	"inventory-app/models"

	"github.com/go-chi/chi/v5"
)

// const uploadDir = "uploads" // 削除：handlers/images.goで定義済み

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

func (e *Env) MarketList(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	if user.DepartmentID == nil {
		http.Error(w, "部門未所属", http.StatusForbidden)
		return
	}

	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	items, err := models.ListItems(e.DB, models.ItemFilter{
		DepartmentID: *user.DepartmentID,
		Status:       "market",
		Query:        query,
		Limit:        perPage + 1,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasMore := len(items) > perPage
	if hasMore {
		items = items[:perPage]
	}

	imageMap := collectImages(e, items)

	data := map[string]any{
		"User":     user,
		"Items":    items,
		"ImageMap": imageMap,
		"Query":    query,
		"Page":     page,
		"HasMore":  hasMore,
	}

	if r.Header.Get("HX-Request") == "true" {
		e.renderPartial(w, "item_list_partial.html", data)
		return
	}
	e.render(w, "market.html", data)
}

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

func (e *Env) CreateItemForm(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	e.render(w, "item_form.html", map[string]any{"User": user})
}

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

func (e *Env) PutOnMarket(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, _ := strconv.Atoi(r.FormValue("item_id"))

	if err := models.PutItemOnMarket(e.DB, itemID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "マーケットに出品しました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

func (e *Env) WithdrawFromMarket(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, _ := strconv.Atoi(r.FormValue("item_id"))

	if err := models.WithdrawItem(e.DB, itemID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	triggerToast(w, "出品を取り下げました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

func (e *Env) ApplyForItem(w http.ResponseWriter, r *http.Request) {
	user := mw.CurrentUser(r)
	itemID, _ := strconv.Atoi(r.FormValue("item_id"))

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
	// Allow title update as well if needed. For now, only description.

	if err := models.UpdateItemDescription(e.DB, itemID, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle image uploads
	// First, count existing images
	existingImages, err := models.ListItemImages(e.DB, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currentImageCount := len(existingImages)
	maxImages := 5

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
		os.Remove(filepath.Join(uploadDir, p))
	}
	os.Remove(filepath.Join(uploadDir, strconv.Itoa(itemID)))

	triggerToast(w, "アイテムを削除しました")
	w.Header().Set("HX-Redirect", "/my-items")
	w.WriteHeader(http.StatusOK)
}

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

	os.Remove(filepath.Join(uploadDir, filePath))
	triggerToast(w, "画像を削除しました")
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}
