package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const (
    maxNoticeTitle = 120
    maxNoticeBody  = 2000
)

type NoticeForm struct {
    Title    string
    Body     string
    Audience string // all, teacher, student, parent or class:<id>
    Pinned   bool
}

type NoticeFormData struct {
    Editing bool
    ID      int
    Action  string
    Form    NoticeForm
    Classes []models.Class
}

func readNoticeForm(r *http.Request) NoticeForm {
    return NoticeForm{
        Title:    strings.TrimSpace(r.PostFormValue("title")),
        Body:     strings.TrimSpace(r.PostFormValue("body")),
        Audience: r.PostFormValue("audience"),
        Pinned:   r.PostFormValue("pinned") == "1",
    }
}

// validateNotice checks the form and turns the audience choice into a notice input.
func (a *App) validateNotice(f NoticeForm) (models.NoticeInput, string) {
    var in models.NoticeInput

    if f.Title == "" || utf8.RuneCountInString(f.Title) > maxNoticeTitle {
        return in, "Title is required (max 120 characters)"
    }
    if f.Body == "" || utf8.RuneCountInString(f.Body) > maxNoticeBody {
        return in, "Message is required (max 2000 characters)"
    }

    audience, classID := "", 0
    switch f.Audience {
    case "all", "teacher", "student", "parent":
        audience = f.Audience
    default:
        if strings.HasPrefix(f.Audience, "class:") {
            id, err := strconv.Atoi(strings.TrimPrefix(f.Audience, "class:"))
            if err == nil && id > 0 {
                if _, err := models.GetClass(a.DB, id); err == nil {
                    audience, classID = "class", id
                }
            }
        }
    }
    if audience == "" {
        return in, "Please choose who should see this notice"
    }

    return models.NoticeInput{
        Title:    f.Title,
        Body:     f.Body,
        Audience: audience,
        ClassID:  classID,
        Pinned:   f.Pinned,
    }, ""
}

func (a *App) renderNoticeForm(w http.ResponseWriter, r *http.Request, status int, data NoticeFormData, msg string) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data.Classes = classes

    title := "Post a notice"
    data.Action = "/admin/notices"
    if data.Editing {
        title = "Edit notice"
        data.Action = "/admin/notices/" + strconv.Itoa(data.ID)
    }
    a.render(w, r, "notice_form", status, PageData{Title: title, Error: msg, Data: data})
}

// NoticeList shows the notices meant for the logged-in user (every notice for an admin).
func (a *App) NoticeList(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    list, err := models.ListNoticesForUser(a.DB, u.Role, u.ID, 200)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "notices", http.StatusOK, PageData{Title: "Notices", Data: list})
}

func (a *App) NoticeNewForm(w http.ResponseWriter, r *http.Request) {
    a.renderNoticeForm(w, r, http.StatusOK, NoticeFormData{Form: NoticeForm{Audience: "all"}}, "")
}

func (a *App) NoticeCreate(w http.ResponseWriter, r *http.Request) {
    f := readNoticeForm(r)
    data := NoticeFormData{Form: f}

    in, msg := a.validateNotice(f)
    if msg != "" {
        a.renderNoticeForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.CreateNotice(a.DB, in); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/notices", http.StatusSeeOther)
}

func (a *App) NoticeEditForm(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    n, err := models.GetNotice(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderNoticeForm(w, r, http.StatusOK, NoticeFormData{
        Editing: true,
        ID:      n.ID,
        Form:    NoticeForm{Title: n.Title, Body: n.Body, Audience: n.AudienceValue(), Pinned: n.Pinned},
    }, "")
}

func (a *App) NoticeUpdate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if _, err := models.GetNotice(a.DB, id); err != nil {
        if errors.Is(err, models.ErrNotFound) {
            http.NotFound(w, r)
            return
        }
        a.serverError(w, err)
        return
    }

    f := readNoticeForm(r)
    data := NoticeFormData{Editing: true, ID: id, Form: f}

    in, msg := a.validateNotice(f)
    if msg != "" {
        a.renderNoticeForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.UpdateNotice(a.DB, id, in); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/notices", http.StatusSeeOther)
}

func (a *App) NoticeDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteNotice(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/notices", http.StatusSeeOther)
}
