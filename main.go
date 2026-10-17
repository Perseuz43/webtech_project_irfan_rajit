package main

import (
    "html/template"
    "log"
    "net/http"

    "github.com/Perseuz43/webtech_project_irfan_rajit/config"
    "github.com/Perseuz43/webtech_project_irfan_rajit/database"
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

    tmpl := template.Must(template.ParseFiles("templates/home.html"))

    mux := http.NewServeMux()

    // serve CSS/JS files
    mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    // home page
    mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
        data := map[string]string{"Title": "School Management System"}
        if err := tmpl.Execute(w, data); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    log.Println("Server running on http://localhost:" + cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
