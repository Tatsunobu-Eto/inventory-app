package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	_ "time/tzdata" // タイムゾーンデータをバイナリに埋め込み（オフライン環境対応）

	"inventory-app/handlers"
	mw "inventory-app/middleware"
	"inventory-app/models"

	"github.com/go-chi/chi/v5"
	chiMw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

//go:embed migrations/*.sql
var migrationsFS embed.FS

// main はアプリケーションのエントリポイント。
// .env 読み込み → DB接続 → マイグレーション → 初期ユーザー作成 → HTTPサーバー起動 の順で処理する。
func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("DB open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping: %v", err)
	}

	// DBマイグレーションを実行する（未適用のものだけ実行される）
	runMigrations(db)

	// sysadmin が1人もいない場合は初期ユーザーを作成する
	seedSysadmin(db)

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY environment variable is not set. Please set a strong random key.")
	}
	store := sessions.NewCookieStore([]byte(sessionKey))

	tmplFS, _ := fs.Sub(templateFS, "templates")
	env := handlers.NewEnv(db, store, tmplFS)

	r := chi.NewRouter()
	r.Use(chiMw.Logger)
	r.Use(chiMw.Recoverer)

	// 静的ファイル（CSS・JS）をバイナリ埋め込みから配信する
	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// アップロードされた画像をディスクから配信する
	os.MkdirAll("uploads", 0o755)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// 認証不要の公開ルート
	r.Get("/login", env.LoginPage)
	r.Post("/login", env.LoginPost)
	r.Get("/logout", env.Logout)

	// 要ログインのルート（Auth ミドルウェアでセッションを検証する）
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(db, store))

		r.Get("/", env.Dashboard)

		// 一般ユーザー・管理者共通のアイテム操作ルート
		r.Get("/market", env.MarketList)
		r.Get("/my-items", env.MyItems)
		r.Get("/items/new", env.CreateItemForm)
		r.Post("/items/new", env.CreateItemPost)
		r.Post("/items/put-on-market", env.PutOnMarket)
		r.Post("/items/withdraw", env.WithdrawFromMarket)
		r.Post("/items/apply", env.ApplyForItem)

		// 取引履歴・応募承認
		r.Get("/transactions", env.Transactions)
		r.Get("/apply/success", env.ApplySuccess)
		r.Post("/transactions/{transaction_id}/approve", env.ApproveTransaction)
		r.Post("/transactions/{transaction_id}/reject", env.RejectTransaction)

		// パスワード変更
		r.Get("/profile/password", env.PasswordChangePage)
		r.Post("/profile/password", env.PasswordChangePost)

		// アイテム詳細・更新・削除
		r.Get("/items/{item_id}", env.ItemDetail)
		r.Post("/items/{item_id}", env.UpdateItemPost)
		r.Post("/items/{item_id}/delete", env.DeleteItem)
		r.Post("/items/images/{image_id}/delete", env.DeleteItemImage)

		// 部門管理者（admin）専用ルート
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin"))
			r.Get("/admin/users", env.AdminUsers)
			r.Post("/admin/users", env.AdminCreateUser)
			r.Post("/admin/users/{user_id}/delete", env.AdminDeleteUser)
			r.Post("/admin/users/{user_id}/reset-password", env.AdminResetPassword)
			r.Post("/admin/users/{user_id}/transfer", env.AdminTransferUser)
			r.Get("/admin/items", env.AdminDeptItems)
			r.Get("/admin/items/new", env.AdminCreateItemForm)
			r.Post("/admin/items", env.AdminCreateItem)
		})

		// システム管理者（sysadmin）専用ルート
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("sysadmin"))
			r.Get("/sysadmin/departments", env.SysAdminDepartments)
			r.Post("/sysadmin/departments", env.SysAdminCreateDepartment)
			r.Post("/sysadmin/admins", env.SysAdminCreateAdmin)
			r.Get("/sysadmin/items", env.SysAdminAllItems)
			r.Post("/sysadmin/users/{user_id}/delete", env.SysAdminDeleteUser)
			r.Post("/sysadmin/users/{user_id}/reset-password", env.SysAdminResetPassword)
			r.Post("/sysadmin/users/{user_id}/promote", env.SysAdminPromoteToAdmin)
				r.Post("/sysadmin/users/{user_id}/demote", env.SysAdminDemoteToUser)
			r.Post("/sysadmin/users/{user_id}/transfer", env.SysAdminTransferUser)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	fmt.Printf("Server starting on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// runMigrations はバイナリに埋め込まれたSQLファイルを使ってDBマイグレーションを実行する。
// 既に適用済みの場合は何もしない（ErrNoChange を正常として扱う）。
func runMigrations(db *sql.DB) {
	migFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		log.Fatalf("migration fs: %v", err)
	}
	source, err := iofs.New(migFS, ".")
	if err != nil {
		log.Fatalf("migration source: %v", err)
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("migration driver: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		log.Fatalf("migration init: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration up: %v", err)
	}
	log.Println("Migrations applied successfully")
}

// seedSysadmin はシステム管理者が1人もいない場合に、環境変数から初期 sysadmin を作成する。
// 初回起動時のみ実行される想定。
func seedSysadmin(db *sql.DB) {
	count, err := models.CountSysadmins(db)
	if err != nil {
		log.Fatalf("count sysadmins: %v", err)
	}
	if count > 0 {
		return
	}

	username := os.Getenv("INIT_SYSADMIN_USER")
	password := os.Getenv("INIT_SYSADMIN_PASS")
	if username == "" || password == "" {
		log.Println("No sysadmin found and INIT_SYSADMIN_USER/PASS not set, skipping seed")
		return
	}

	_, err = models.CreateUser(db, nil, username, password, "System Admin", "sysadmin")
	if err != nil {
		log.Fatalf("seed sysadmin: %v", err)
	}
	log.Printf("Initial sysadmin '%s' created", username)
}
