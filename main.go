package main

import (
    "html/template"
    "log"
    "net/http"
)

func main() {
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

    log.Println("Server running on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
