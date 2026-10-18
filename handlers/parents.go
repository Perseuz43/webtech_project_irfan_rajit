package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type ParentForm struct {
    Name     string
    Phone    string
    Username string
    Password string
}

type ParentFormData struct {
    Editing  bool
    ID       int
    Action   string
    Username string
    Form     ParentForm
    Children []models.ChildOption
}

func readParentForm(r *http.Request) ParentForm {
    return ParentForm{
        Name:     strings.TrimSpace(r.PostFormValue("name")),
        Phone:    strings.TrimSpace(r.PostFormValue("phone")),
        Username: strings.TrimSpace(r.PostFormValue("username")),
        Password: r.PostFormValue("password"),
    }
}

func validateParent(f ParentForm, editing bool) (models.ParentInput, string) {
    var in models.ParentInput

    if f.Name == "" || utf8.RuneCountInString(f.Name) > 100 {
        return in, "Name is required (max 100 characters)"
    }
    if !validPhone(f.Phone) {
        return in, "Phone may only contain digits, spaces, + and - (max 20 characters)"
    }
    if !editing {
        if msg := validateCredentials(f.Username, f.Password); msg != "" {
            return in, msg
        }
    }
    return models.ParentInput{
        Name:     f.Name,
        Phone:    f.Phone,
        Username: f.Username,
        Password: f.Password,
    }, ""
}

func (a *App) renderParentForm(w http.ResponseWriter, r *http.Request, status int, data ParentFormData, msg string) {
    title := "Add parent"
    data.Action = "/admin/parents"
    if data.Editing {
        title = "Edit parent"
        data.Action = "/admin/parents/" + strconv.Itoa(data.ID)
        children, err := models.ListChildOptions(a.DB, data.ID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        data.Children = children
    }
    a.render(w, r, "parent_form", status, PageData{Title: title, Error: msg, Data: data})
}

func (a *App) ParentList(w http.ResponseWriter, r *http.Request) {
    parents, err := models.ListParents(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "parents", http.StatusOK, PageData{Title: "Parents", Data: parents})
}

func (a *App) ParentNewForm(w http.ResponseWriter, r *http.Request) {
    a.renderParentForm(w, r, http.StatusOK, ParentFormData{}, "")
}

func (a *App) ParentCreate(w http.ResponseWriter, r *http.Request) {
    f := readParentForm(r)
    data := ParentFormData{Form: f}

    in, msg := validateParent(f, false)
    if msg != "" {
        a.renderParentForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.CreateParent(a.DB, in); err != nil {
        if models.IsDuplicate(err) {
            a.renderParentForm(w, r, http.StatusConflict, data, "That username is already taken")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

func (a *App) ParentEditForm(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    p, err := models.GetParent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderParentForm(w, r, http.StatusOK, ParentFormData{
        Editing:  true,
        ID:       p.ID,
        Username: p.Username,
        Form:     ParentForm{Name: p.Name, Phone: p.Phone},
    }, "")
}

func (a *App) ParentUpdate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    p, err := models.GetParent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }

    f := readParentForm(r)
    data := ParentFormData{Editing: true, ID: p.ID, Username: p.Username, Form: f}

    in, msg := validateParent(f, true)
    if msg != "" {
        a.renderParentForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.UpdateParent(a.DB, id, in); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// ParentChildren saves which students are the parent's children.
func (a *App) ParentChildren(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if _, err := models.GetParent(a.DB, id); err != nil {
        if errors.Is(err, models.ErrNotFound) {
            http.NotFound(w, r)
            return
        }
        a.serverError(w, err)
        return
    }
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    var ids []int
    for _, s := range r.PostForm["student_ids"] {
        if n, err := strconv.Atoi(s); err == nil && n > 0 {
            ids = append(ids, n)
        }
    }
    if err := models.SetParentChildren(a.DB, id, ids); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/parents/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
}

func (a *App) ParentDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteParent(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}
