package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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

	// Run migrations
	runMigrations(db)

	// Seed initial sysadmin
	seedSysadmin(db)

	// Start background job for 90-day expiry
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			ids, err := models.ExpireMarketItems(db)
			if err != nil {
				log.Printf("expire job error: %v", err)
			} else if len(ids) > 0 {
				log.Printf("expired %d market items", len(ids))
				cleanupItemFiles(db, ids)
			}
			<-ticker.C
		}
	}()

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		sessionKey = "default-dev-key-change-me!!"
	}
	store := sessions.NewCookieStore([]byte(sessionKey))

	tmplFS, _ := fs.Sub(templateFS, "templates")
	env := handlers.NewEnv(db, store, tmplFS)

	r := chi.NewRouter()
	r.Use(chiMw.Logger)
	r.Use(chiMw.Recoverer)

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Uploaded images
	os.MkdirAll("uploads", 0o755)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// Public routes
	r.Get("/login", env.LoginPage)
	r.Post("/login", env.LoginPost)
	r.Get("/logout", env.Logout)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(db, store))

		r.Get("/", env.Dashboard)

		// User / Admin item routes
		r.Get("/market", env.MarketList)
		r.Get("/my-items", env.MyItems)
		r.Get("/items/new", env.CreateItemForm)
		r.Post("/items/new", env.CreateItemPost)
		r.Post("/items/put-on-market", env.PutOnMarket)
		r.Post("/items/withdraw", env.WithdrawFromMarket)
		r.Post("/items/apply", env.ApplyForItem)

		// Transaction history
		r.Get("/transactions", env.Transactions)

		// Password change
		r.Get("/profile/password", env.PasswordChangePage)
		r.Post("/profile/password", env.PasswordChangePost)

		// New item detail and update routes
		r.Get("/items/{item_id}", env.ItemDetail)
		r.Post("/items/{item_id}", env.UpdateItemPost)
		r.Post("/items/{item_id}/delete", env.DeleteItem)
		r.Post("/items/images/{image_id}/delete", env.DeleteItemImage)

		// Department admin routes
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

		// System admin routes
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

// cleanupItemFiles deletes upload files for the given item IDs from disk.
// Called after items are expired or deleted so no orphan files remain.
func cleanupItemFiles(db *sql.DB, itemIDs []int) {
	imgMap, err := models.ListItemImagesByItems(db, itemIDs)
	if err != nil {
		log.Printf("cleanup: failed to list images: %v", err)
		return
	}
	for itemID, imgs := range imgMap {
		for _, img := range imgs {
			path := filepath.Join("uploads", img.FilePath)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Printf("cleanup: remove %s: %v", path, err)
			}
		}
		dir := filepath.Join("uploads", strconv.Itoa(itemID))
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			log.Printf("cleanup: remove dir %s: %v", dir, err)
		}
	}
}

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
