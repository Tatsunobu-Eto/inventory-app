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

// Env は全ハンドラが共有する依存オブジェクトをまとめた構造体。
type Env struct {
	DB    *sql.DB
	Store sessions.Store
	tmpls map[string]*template.Template // テンプレート名→コンパイル済みテンプレートのマップ
}

// perPage は1ページあたりの表示件数。
const perPage = 10

// funcs はHTMLテンプレート内で使用できるカスタム関数の一覧。
var funcs = template.FuncMap{
	// dict: キーと値を交互に渡してマップを作る（テンプレートで部分的にデータを渡す際に使用）
	"dict": func(pairs ...any) map[string]any {
		m := make(map[string]any, len(pairs)/2)
		for i := 0; i < len(pairs)-1; i += 2 {
			m[pairs[i].(string)] = pairs[i+1]
		}
		return m
	},
	// add: 2つの整数を足す（ページネーションの番号計算に使用）
	"add": func(a, b int) int { return a + b },
	// seq: start から end までの整数スライスを生成する（ページ番号リスト生成に使用）
	"seq": func(start, end int) []int {
		s := make([]int, 0, end-start+1)
		for i := start; i <= end; i++ {
			s = append(s, i)
		}
		return s
	},
	// images: アイテムIDに対応する画像一覧をマップから取り出す
	"images": func(imageMap map[int][]models.ItemImage, itemID int) []models.ItemImage {
		return imageMap[itemID]
	},
}

// NewEnv はテンプレートを事前にコンパイルしてハンドラ共有オブジェクトを生成する。
// テンプレートはアプリ起動時に一度だけコンパイルされるため、リクエストごとの処理が高速になる。
func NewEnv(db *sql.DB, store sessions.Store, tmplFS fs.FS) *Env {
	tmpls := make(map[string]*template.Template)

	// layout.html を使うページテンプレートを事前コンパイルする
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

	// レイアウトなしのスタンドアロンページ（ログイン画面）
	tmpls["login.html"] = template.Must(
		template.New("").Funcs(funcs).ParseFS(tmplFS, "login.html"),
	)

	// HTMX部分更新用のパーシャルテンプレート
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

// render はフルページHTMLをレスポンスに書き出す。
// 通常は layout.html をエントリポイントとしてレンダリングし、ログインページのみ単独で描画する。
func (e *Env) render(w http.ResponseWriter, name string, data map[string]any) {
	t, ok := e.tmpls[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// ログインページはレイアウトなし、それ以外は layout.html から {{template "content" .}} を呼び出す
	execName := "layout.html"
	if name == "login.html" {
		execName = "login.html"
	}
	if err := t.ExecuteTemplate(w, execName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderPartial はHTMXからの部分更新リクエスト用にパーシャルテンプレートを描画する。
// フルページレイアウトは使わず、指定したテンプレート断片のみを返す。
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

// triggerToast はHTMXの HX-Trigger ヘッダにトースト通知メッセージをセットする。
// フロントエンドの Alpine.js が "showMessage" イベントを受け取り、トーストを表示する。
// メッセージ内の日本語・特殊文字はUnicodeエスケープしてJSONの不正を防ぐ。
func triggerToast(w http.ResponseWriter, msg string) {
	var sb strings.Builder
	for _, r := range msg {
		switch {
		case r == '"' || r == '\\':
			// JSON文字列を壊す特殊文字をエスケープ
			sb.WriteRune('\\')
			sb.WriteRune(r)
		case r < 0x20:
			// 制御文字をUnicodeエスケープ
			fmt.Fprintf(&sb, `\u%04x`, r)
		case r > 0x7E:
			// ASCII範囲外（日本語など）をUnicodeエスケープ
			fmt.Fprintf(&sb, `\u%04x`, r)
		default:
			sb.WriteRune(r)
		}
	}
	w.Header().Set("HX-Trigger", `{"showMessage":"`+sb.String()+`"}`)
}
