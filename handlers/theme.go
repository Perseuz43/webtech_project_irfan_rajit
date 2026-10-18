package handlers

import (
    "net/http"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const themeCookie = "theme"

// cleanTheme returns the value only if it is a known theme, otherwise "" (follow the system setting).
func cleanTheme(t string) string {
    if t == "light" || t == "dark" {
        return t
    }
    return ""
}

// themeFor decides which theme to render: the logged-in user's choice, or the browser cookie.
func (a *App) themeFor(r *http.Request) string {
    if a.currentUser(r) != nil {
        return cleanTheme(a.Sessions.GetString(r.Context(), "theme"))
    }
    if c, err := r.Cookie(themeCookie); err == nil {
        return cleanTheme(c.Value)
    }
    return ""
}

// loadTheme runs at login. It returns the account's saved theme. If the account has none yet,
// it adopts the theme chosen on the login page (from the cookie) and saves it.
func (a *App) loadTheme(r *http.Request, userID int) string {
    theme, err := models.GetTheme(a.DB, userID)
    if err != nil {
        return ""
    }
    if theme = cleanTheme(theme); theme != "" {
        return theme
    }
    if c, err := r.Cookie(themeCookie); err == nil {
        if t := cleanTheme(c.Value); t != "" {
            _ = models.SetTheme(a.DB, userID, t)
            return t
        }
    }
    return ""
}

// SetTheme saves the choice for the logged-in user and remembers it in a cookie for the login page.
func (a *App) SetTheme(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    theme := r.PostFormValue("theme")
    if theme != "light" && theme != "dark" {
        http.Error(w, "Invalid theme", http.StatusBadRequest)
        return
    }

    if u := a.currentUser(r); u != nil {
        if err := models.SetTheme(a.DB, u.ID, theme); err != nil {
            a.serverError(w, err)
            return
        }
        a.Sessions.Put(r.Context(), "theme", theme)
    }

    http.SetCookie(w, &http.Cookie{
        Name:     themeCookie,
        Value:    theme,
        Path:     "/",
        MaxAge:   365 * 24 * 60 * 60,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })
    w.WriteHeader(http.StatusNoContent)
}
