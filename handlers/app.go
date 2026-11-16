package handlers

import (
    "bytes"
    "database/sql"
    "html/template"
    "log"
    "net/http"
    "path/filepath"
    "strings"

    "github.com/alexedwards/scs/v2"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
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
    Title  string
    User   *SessionUser
    Error  string
    Data   any
    Theme  string
    Path   string
    School SchoolInfo

    Years       []models.AcademicYear // admins: every academic year, for the menu bar
    WorkingYear *models.AcademicYear  // admins: the year they are viewing (nil = all years)
}

// New parses every page in the templates folder together with the shared layout.
// A page named templates/exams.html is rendered with a.render(..., "exams", ...).
func New(db *sql.DB, sm *scs.SessionManager) (*App, error) {
    funcs := template.FuncMap{"hasPrefix": strings.HasPrefix}

    files, err := filepath.Glob("templates/*.html")
    if err != nil {
        return nil, err
    }

    tmpls := make(map[string]*template.Template)
    for _, f := range files {
        base := filepath.Base(f)
        if base == "layout.html" {
            continue
        }
        t, err := template.New("layout.html").Funcs(funcs).ParseFiles("templates/layout.html", f)
        if err != nil {
            return nil, err
        }
        tmpls[strings.TrimSuffix(base, ".html")] = t
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
    data.Theme = a.themeFor(r)
    data.Path = r.URL.Path
    data.School = a.schoolInfo()
    if data.User != nil && data.User.Role == "admin" {
        data.Years, data.WorkingYear = a.workingYear(r)
    }

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
