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
    admin := func(h http.HandlerFunc) http.Handler { return requireAdmin(h) }

    mux := http.NewServeMux()

    // serve CSS/JS files
    mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    // public pages
    mux.HandleFunc("GET /{$}", app.Home)
    mux.HandleFunc("GET /login", app.LoginForm)
    mux.HandleFunc("POST /login", app.Login)
    mux.HandleFunc("POST /logout", app.Logout)

    // logged-in pages
    mux.Handle("GET /dashboard", requireLogin(http.HandlerFunc(app.Dashboard)))

    // admin pages
    mux.Handle("GET /admin", admin(app.AdminHome))

    mux.Handle("GET /admin/classes", admin(app.ClassList))
    mux.Handle("POST /admin/classes", admin(app.ClassCreate))
    mux.Handle("GET /admin/classes/{id}", admin(app.ClassDetail))
    mux.Handle("POST /admin/classes/{id}/rename", admin(app.ClassRename))
    mux.Handle("POST /admin/classes/{id}/delete", admin(app.ClassDelete))
    mux.Handle("POST /admin/classes/{id}/sections", admin(app.SectionCreate))
    mux.Handle("POST /admin/classes/{id}/sections/{sid}/delete", admin(app.SectionDelete))
    mux.Handle("POST /admin/classes/{id}/subjects", admin(app.SubjectCreate))
    mux.Handle("POST /admin/classes/{id}/subjects/{sid}/delete", admin(app.SubjectDelete))

    mux.Handle("GET /admin/students", admin(app.StudentList))
    mux.Handle("GET /admin/students/new", admin(app.StudentNewForm))
    mux.Handle("POST /admin/students", admin(app.StudentCreate))
    mux.Handle("GET /admin/students/{id}/edit", admin(app.StudentEditForm))
    mux.Handle("POST /admin/students/{id}", admin(app.StudentUpdate))
    mux.Handle("POST /admin/students/{id}/delete", admin(app.StudentDelete))

    mux.Handle("GET /admin/teachers", admin(app.TeacherList))
    mux.Handle("GET /admin/teachers/new", admin(app.TeacherNewForm))
    mux.Handle("POST /admin/teachers", admin(app.TeacherCreate))
    mux.Handle("GET /admin/teachers/{id}/edit", admin(app.TeacherEditForm))
    mux.Handle("POST /admin/teachers/{id}", admin(app.TeacherUpdate))
    mux.Handle("POST /admin/teachers/{id}/subjects", admin(app.TeacherSubjects))
    mux.Handle("POST /admin/teachers/{id}/delete", admin(app.TeacherDelete))

    // block cross-site form submissions (CSRF protection)
    cop := http.NewCrossOriginProtection()

    log.Println("Server running on http://localhost:" + cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, sm.LoadAndSave(cop.Handler(mux))))
}
