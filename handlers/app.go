package handlers

import (
    "bytes"
    "database/sql"
    "html/template"
    "log"
    "net/http"

    "github.com/alexedwards/scs/v2"
)

type App struct {
    DB        *sql.DB
    Sessions  *scs.SessionManager
    Templates map[string]*template.Template
}

type SessionUser struct {
    ID       int
    Username string
    Role     string
}

type PageData struct {
    Title string
    User  *SessionUser
    Error string
}

// New parses every page together with the shared layout.
func New(db *sql.DB, sm *scs.SessionManager) (*App, error) {
    pages := []string{"home", "login", "dashboard", "admin"}
    tmpls := make(map[string]*template.Template)
    for _, p := range pages {
        t, err := template.ParseFiles("templates/layout.html", "templates/"+p+".html")
        if err != nil {
            return nil, err
        }
        tmpls[p] = t
    }
    return &App{DB: db, Sessions: sm, Templates: tmpls}, nil
}

func (a *App) currentUser(r *http.Request) *SessionUser {
    id := a.Sessions.GetInt(r.Context(), "user_id")
    if id == 0 {
        return nil
    }
    return &SessionUser{
        ID:       id,
        Username: a.Sessions.GetString(r.Context(), "username"),
        Role:     a.Sessions.GetString(r.Context(), "role"),
    }
}

func (a *App) render(w http.ResponseWriter, r *http.Request, page string, status int, data PageData) {
    tmpl, ok := a.Templates[page]
    if !ok {
        http.Error(w, "template not found", http.StatusInternalServerError)
        return
    }
    data.User = a.currentUser(r)

    var buf bytes.Buffer
    if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
        log.Println("template error:", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    buf.WriteTo(w)
}
