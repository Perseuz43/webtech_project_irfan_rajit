package handlers

import (
    "errors"
    "log"
    "net/http"
    "strings"

    "golang.org/x/crypto/bcrypt"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
    if a.currentUser(r) != nil {
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
        return
    }
    a.render(w, r, "home", http.StatusOK, PageData{Title: "School Management System"})
}

func (a *App) LoginForm(w http.ResponseWriter, r *http.Request) {
    if a.currentUser(r) != nil {
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
        return
    }
    a.render(w, r, "login", http.StatusOK, PageData{Title: "Login"})
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    username := strings.TrimSpace(r.PostFormValue("username"))
    password := r.PostFormValue("password")

    user, err := models.GetUserByUsername(a.DB, username)
    if err == nil {
        err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    }
    if err != nil {
        if errors.Is(err, models.ErrNotFound) || errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
            // same message for both cases, so attackers cannot tell which usernames exist
            a.render(w, r, "login", http.StatusUnauthorized, PageData{Title: "Login", Error: "Invalid username or password"})
            return
        }
        log.Println("login error:", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // new session ID on login protects against session fixation
    if err := a.Sessions.RenewToken(r.Context()); err != nil {
        log.Println("session error:", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    a.Sessions.Put(r.Context(), "user_id", user.ID)
    a.Sessions.Put(r.Context(), "username", user.Username)
    a.Sessions.Put(r.Context(), "role", user.Role)

    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
    if err := a.Sessions.Destroy(r.Context()); err != nil {
        log.Println("logout error:", err)
    }
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}


func (a *App) AdminHome(w http.ResponseWriter, r *http.Request) {
    a.render(w, r, "admin", http.StatusOK, PageData{Title: "Admin Area"})
}
