package main

import (
    "log"
    "net/http"
    "time"

    "github.com/alexedwards/scs/v2"

    "github.com/Perseuz43/webtech_project_irfan_rajit/config"
    "github.com/Perseuz43/webtech_project_irfan_rajit/database"
    "github.com/Perseuz43/webtech_project_irfan_rajit/handlers"
    "github.com/Perseuz43/webtech_project_irfan_rajit/middleware"
    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

func main() {
    cfg := config.Load()

    db, err := database.Connect(cfg.DBPath)
    if err != nil {
        log.Fatal("database connection failed: ", err)
    }
    defer db.Close()

    if err := database.Migrate(db); err != nil {
        log.Fatal("database migration failed: ", err)
    }
    log.Println("Database connected and migrated:", cfg.DBPath)

    if err := models.SeedAdmin(db, cfg.AdminPassword); err != nil {
        log.Fatal("creating default admin failed: ", err)
    }

    sm := scs.New()
    sm.Lifetime = 12 * time.Hour

    app, err := handlers.New(db, sm)
    if err != nil {
        log.Fatal("loading templates failed: ", err)
    }

    requireLogin := middleware.RequireLogin(sm)
    requireAdmin := middleware.RequireRole(sm, "admin")

    mux := http.NewServeMux()

    // serve CSS/JS files
    mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    // public pages
    mux.HandleFunc("GET /{$}", app.Home)
    mux.HandleFunc("GET /login", app.LoginForm)
    mux.HandleFunc("POST /login", app.Login)
    mux.HandleFunc("POST /logout", app.Logout)

    // protected pages
    mux.Handle("GET /dashboard", requireLogin(http.HandlerFunc(app.Dashboard)))
    mux.Handle("GET /admin", requireAdmin(http.HandlerFunc(app.AdminHome)))

    log.Println("Server running on http://localhost:" + cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, sm.LoadAndSave(mux)))
}
