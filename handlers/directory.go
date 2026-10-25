package handlers

import (
    "net/http"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const maxSearchLen = 50

type DirectoryData struct {
    Classes   []models.Class
    Sections  []models.Section
    ClassID   int
    SectionID int
    Query     string
    Entries   []models.DirectoryEntry
}

// StudentDirectory lists students with their class and parent contacts.
// Admins see every student; teachers only see students in classes where they teach.
func (a *App) StudentDirectory(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    q := r.URL.Query()

    classes, sections, err := a.classChoices(r)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data := DirectoryData{Classes: classes, Sections: sections}
    msg := ""

    search := strings.TrimSpace(q.Get("q"))
    if utf8.RuneCountInString(search) > maxSearchLen {
        msg = "The search text is too long (max 50 characters)"
        search = ""
    }
    data.Query = search

    if msg == "" && q.Get("class_id") != "" {
        classID, sectionID, classMsg, err := a.checkClassSection(classes, q.Get("class_id"), q.Get("section_id"))
        if err != nil {
            a.serverError(w, err)
            return
        }
        if classMsg != "" {
            msg = classMsg
        } else {
            data.ClassID, data.SectionID = classID, sectionID
        }
    }

    status := http.StatusOK
    if msg != "" {
        status = http.StatusBadRequest
    } else {
        entries, err := models.ListDirectory(a.DB, u.Role, u.ID, data.ClassID, data.SectionID, data.Query)
        if err != nil {
            a.serverError(w, err)
            return
        }
        data.Entries = entries
    }

    a.render(w, r, "directory", status, PageData{Title: "Student directory", Error: msg, Data: data})
}
