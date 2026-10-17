package handlers

import (
    "errors"
    "log"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const maxNameLen = 50

type ClassDetailData struct {
    Class    *models.Class
    Sections []models.Section
    Subjects []models.Subject
}

// pathInt reads a numeric value from the URL, e.g. /admin/classes/{id}.
func pathInt(r *http.Request, name string) (int, bool) {
    n, err := strconv.Atoi(r.PathValue(name))
    return n, err == nil && n > 0
}

// cleanName reads and validates the "name" form field. The second value is an error message.
func cleanName(r *http.Request) (string, string) {
    if err := r.ParseForm(); err != nil {
        return "", "Bad request"
    }
    name := strings.TrimSpace(r.PostFormValue("name"))
    if name == "" {
        return "", "Name cannot be empty"
    }
    if utf8.RuneCountInString(name) > maxNameLen {
        return "", "Name is too long (max 50 characters)"
    }
    return name, ""
}

func (a *App) serverError(w http.ResponseWriter, err error) {
    log.Println("server error:", err)
    http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// ---------- classes ----------

func (a *App) renderClassList(w http.ResponseWriter, r *http.Request, status int, msg string) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "classes", status, PageData{Title: "Classes", Error: msg, Data: classes})
}

func (a *App) ClassList(w http.ResponseWriter, r *http.Request) {
    a.renderClassList(w, r, http.StatusOK, "")
}

func (a *App) ClassCreate(w http.ResponseWriter, r *http.Request) {
    name, msg := cleanName(r)
    if msg != "" {
        a.renderClassList(w, r, http.StatusBadRequest, msg)
        return
    }
    if err := models.CreateClass(a.DB, name); err != nil {
        if models.IsDuplicate(err) {
            a.renderClassList(w, r, http.StatusConflict, "A class with that name already exists")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes", http.StatusSeeOther)
}

func (a *App) renderClassDetail(w http.ResponseWriter, r *http.Request, id, status int, msg string) {
    class, err := models.GetClass(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    sections, err := models.ListSections(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    subjects, err := models.ListSubjects(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "class_detail", status, PageData{
        Title: class.Name,
        Error: msg,
        Data:  ClassDetailData{Class: class, Sections: sections, Subjects: subjects},
    })
}

func (a *App) ClassDetail(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    a.renderClassDetail(w, r, id, http.StatusOK, "")
}

func (a *App) ClassRename(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    name, msg := cleanName(r)
    if msg != "" {
        a.renderClassDetail(w, r, id, http.StatusBadRequest, msg)
        return
    }
    if err := models.RenameClass(a.DB, id, name); err != nil {
        if models.IsDuplicate(err) {
            a.renderClassDetail(w, r, id, http.StatusConflict, "A class with that name already exists")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) ClassDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteClass(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes", http.StatusSeeOther)
}

// ---------- sections ----------

func (a *App) SectionCreate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    name, msg := cleanName(r)
    if msg != "" {
        a.renderClassDetail(w, r, id, http.StatusBadRequest, msg)
        return
    }
    if err := models.CreateSection(a.DB, id, name); err != nil {
        if models.IsDuplicate(err) {
            a.renderClassDetail(w, r, id, http.StatusConflict, "That section already exists in this class")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) SectionDelete(w http.ResponseWriter, r *http.Request) {
    id, ok1 := pathInt(r, "id")
    sid, ok2 := pathInt(r, "sid")
    if !ok1 || !ok2 {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteSection(a.DB, id, sid); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes/"+strconv.Itoa(id), http.StatusSeeOther)
}

// ---------- subjects ----------

func (a *App) SubjectCreate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    name, msg := cleanName(r)
    if msg != "" {
        a.renderClassDetail(w, r, id, http.StatusBadRequest, msg)
        return
    }
    if err := models.CreateSubject(a.DB, id, name); err != nil {
        if models.IsDuplicate(err) {
            a.renderClassDetail(w, r, id, http.StatusConflict, "That subject already exists in this class")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) SubjectDelete(w http.ResponseWriter, r *http.Request) {
    id, ok1 := pathInt(r, "id")
    sid, ok2 := pathInt(r, "sid")
    if !ok1 || !ok2 {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteSubject(a.DB, id, sid); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/classes/"+strconv.Itoa(id), http.StatusSeeOther)
}
