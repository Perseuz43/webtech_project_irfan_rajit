package middleware

import (
    "net/http"
    "slices"

    "github.com/alexedwards/scs/v2"
)

// RequireLogin redirects visitors who are not logged in to the login page.
func RequireLogin(sm *scs.SessionManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if sm.GetInt(r.Context(), "user_id") == 0 {
                http.Redirect(w, r, "/login", http.StatusSeeOther)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// RequireRole allows only logged-in users whose role is in the given list.
func RequireRole(sm *scs.SessionManager, roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return RequireLogin(sm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := sm.GetString(r.Context(), "role")
            if !slices.Contains(roles, role) {
                http.Error(w, "403 Forbidden: you do not have access to this page", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        }))
    }
}
