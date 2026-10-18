package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type TeacherForm struct {
    Name          string
    Phone         string
    Qualification string
    Username      string
    Password      string
}

type TeacherFormData struct {
    Editing  bool
    ID       int
    Action   string
    Username string
    Form     TeacherForm
    Subjects []models.SubjectOption
}

func readTeacherForm(r *http.Request) TeacherForm {
    return TeacherForm{
        Name:          strings.TrimSpace(r.PostFormValue("name")),
        Phone:         strings.TrimSpace(r.PostFormValue("phone")),
        Qualification: strings.TrimSpace(r.PostFormValue("qualification")),
        Username:      strings.TrimSpace(r.PostFormValue("username")),
        Password:      r.PostFormValue("password"),
    }
}

func validateTeacher(f TeacherForm, editing bool) (models.TeacherInput, string) {
    var in models.TeacherInput

    if f.Name == "" || utf8.RuneCountInString(f.Name) > 100 {
        return in, "Name is required (max 100 characters)"
    }
    if utf8.RuneCountInString(f.Qualification) > 100 {
        return in, "Qualification is too long (max 100 characters)"
    }
    if !validPhone(f.Phone) {
        return in, "Phone may only contain digits, spaces, + and - (max 20 characters)"
    }
    if !editing {
        if msg := validateCredentials(f.Username, f.Password); msg != "" {
            return in, msg
        }
    }
    return models.TeacherInput{
        Name:          f.Name,
        Phone:         f.Phone,
        Qualification: f.Qualification,
        Username:      f.Username,
        Password:      f.Password,
    }, ""
}

func (a *App) renderTeacherForm(w http.ResponseWriter, r *http.Request, status int, data TeacherFormData, msg string) {
    title := "Add teacher"
    data.Action = "/admin/teachers"
    if data.Editing {
        title = "Edit teacher"
        data.Action = "/admin/teachers/" + strconv.Itoa(data.ID)
        subjects, err := models.ListSubjectOptions(a.DB, data.ID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        data.Subjects = subjects
    }
    a.render(w, r, "teacher_form", status, PageData{Title: title, Error: msg, Data: data})
}

func (a *App) TeacherList(w http.ResponseWriter, r *http.Request) {
    teachers, err := models.ListTeachers(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "teachers", http.StatusOK, PageData{Title: "Teachers", Data: teachers})
}

func (a *App) TeacherNewForm(w http.ResponseWriter, r *http.Request) {
    a.renderTeacherForm(w, r, http.StatusOK, TeacherFormData{}, "")
}

func (a *App) TeacherCreate(w http.ResponseWriter, r *http.Request) {
    f := readTeacherForm(r)
    data := TeacherFormData{Form: f}

    in, msg := validateTeacher(f, false)
    if msg != "" {
        a.renderTeacherForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.CreateTeacher(a.DB, in); err != nil {
        if models.IsDuplicate(err) {
            a.renderTeacherForm(w, r, http.StatusConflict, data, "That username is already taken")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/teachers", http.StatusSeeOther)
}

func (a *App) TeacherEditForm(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    t, err := models.GetTeacher(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderTeacherForm(w, r, http.StatusOK, TeacherFormData{
        Editing:  true,
        ID:       t.ID,
        Username: t.Username,
        Form:     TeacherForm{Name: t.Name, Phone: t.Phone, Qualification: t.Qualification},
    }, "")
}

func (a *App) TeacherUpdate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    t, err := models.GetTeacher(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }

    f := readTeacherForm(r)
    data := TeacherFormData{Editing: true, ID: t.ID, Username: t.Username, Form: f}

    in, msg := validateTeacher(f, true)
    if msg != "" {
        a.renderTeacherForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.UpdateTeacher(a.DB, id, in); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/teachers", http.StatusSeeOther)
}

func (a *App) TeacherSubjects(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if _, err := models.GetTeacher(a.DB, id); err != nil {
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
    for _, s := range r.PostForm["subject_ids"] {
        if n, err := strconv.Atoi(s); err == nil && n > 0 {
            ids = append(ids, n)
        }
    }
    if err := models.SetTeacherSubjects(a.DB, id, ids); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/teachers/"+strconv.Itoa(id)+"/edit", http.StatusSeeOther)
}

func (a *App) TeacherDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteTeacher(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/teachers", http.StatusSeeOther)
}
