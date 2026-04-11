package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"inventory-app/models"

	"github.com/gorilla/sessions"
)

// Env holds shared dependencies for all handlers.
type Env struct {
	DB    *sql.DB
	Store sessions.Store
	tmpls map[string]*template.Template
}

const perPage = 10

var funcs = template.FuncMap{
	"dict": func(pairs ...any) map[string]any {
		m := make(map[string]any, len(pairs)/2)
		for i := 0; i < len(pairs)-1; i += 2 {
			m[pairs[i].(string)] = pairs[i+1]
		}
		return m
	},
	"add": func(a, b int) int { return a + b },
	"seq": func(start, end int) []int {
		s := make([]int, 0, end-start+1)
		for i := start; i <= end; i++ {
			s = append(s, i)
		}
		return s
	},
	"images": func(imageMap map[int][]models.ItemImage, itemID int) []models.ItemImage {
		return imageMap[itemID]
	},
}

func NewEnv(db *sql.DB, store sessions.Store, tmplFS fs.FS) *Env {
	tmpls := make(map[string]*template.Template)

	// Pages that use layout.html
	pages := []string{
		"dashboard.html",
		"market.html",
		"my_items.html",
		"item_form.html",
		"sysadmin_departments.html",
		"admin_users.html",
		"admin_item_form.html",
		"admin_dept_items.html",
		"sysadmin_all_items.html",
		"item_detail.html",
		"transactions.html",
		"password_change.html",
	}
	for _, page := range pages {
		tmpls[page] = template.Must(
			template.New("").Funcs(funcs).ParseFS(tmplFS, "layout.html", "item_list_partial.html", "my_items_partial.html", "transactions_partial.html", page),
		)
	}

	// Standalone pages (no layout)
	tmpls["login.html"] = template.Must(
		template.New("").Funcs(funcs).ParseFS(tmplFS, "login.html"),
	)

	// Partials
	tmpls["item_list_partial.html"] = template.Must(
		template.New("").Funcs(funcs).ParseFS(tmplFS, "item_list_partial.html"),
	)
	tmpls["my_items_partial.html"] = template.Must(
		template.New("").Funcs(funcs).ParseFS(tmplFS, "my_items_partial.html"),
	)
	tmpls["transactions_partial.html"] = template.Must(
		template.New("").Funcs(funcs).ParseFS(tmplFS, "transactions_partial.html"),
	)

	return &Env{DB: db, Store: store, tmpls: tmpls}
}

func (e *Env) render(w http.ResponseWriter, name string, data map[string]any) {
	t, ok := e.tmpls[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// For standalone pages (login), execute the page define directly.
	// For layout pages, execute layout.html which calls {{template "content" .}}.
	execName := "layout.html"
	if name == "login.html" {
		execName = "login.html"
	}
	if err := t.ExecuteTemplate(w, execName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (e *Env) renderPartial(w http.ResponseWriter, name string, data any) {
	t, ok := e.tmpls[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func triggerToast(w http.ResponseWriter, msg string) {
	var sb strings.Builder
	for _, r := range msg {
		switch {
		case r == '"' || r == '\\':
			sb.WriteRune('\\')
			sb.WriteRune(r)
		case r < 0x20:
			fmt.Fprintf(&sb, `\u%04x`, r)
		case r > 0x7E:
			fmt.Fprintf(&sb, `\u%04x`, r)
		default:
			sb.WriteRune(r)
		}
	}
	w.Header().Set("HX-Trigger", `{"showMessage":"`+sb.String()+`"}`)
}
