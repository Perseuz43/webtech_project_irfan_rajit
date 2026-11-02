package handlers

import "net/http"

// NotFound shows a friendly 404 page for any address that no other route matches.
func (a *App) NotFound(w http.ResponseWriter, r *http.Request) {
    a.render(w, r, "notfound", http.StatusNotFound, PageData{
        Title: "Page not found",
        Data:  r.URL.Path,
    })
}
