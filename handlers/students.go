package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type StudentForm struct {
    Name         string
    Roll         string
    ClassID      string
    SectionID    string
    GuardianName string
    Phone        string
    Username     string
    Password     string
}

type StudentFormData struct {
    Editing  bool
    ID       int
    Action   string
    Username string
    Form     StudentForm
    Classes  []models.Class
    Sections []models.Section
}

func readStudentForm(r *http.Request) StudentForm {
    return StudentForm{
        Name:         strings.TrimSpace(r.PostFormValue("name")),
        Roll:         strings.TrimSpace(r.PostFormValue("roll")),
        ClassID:      r.PostFormValue("class_id"),
        SectionID:    r.PostFormValue("section_id"),
        GuardianName: strings.TrimSpace(r.PostFormValue("guardian_name")),
        Phone:        strings.TrimSpace(r.PostFormValue("phone")),
        Username:     strings.TrimSpace(r.PostFormValue("username")),
        Password:     r.PostFormValue("password"),
    }
}

// validateStudent checks the form. It returns the cleaned input, or an error message.
func (a *App) validateStudent(f StudentForm, editing bool) (models.StudentInput, string) {
    var in models.StudentInput

    if f.Name == "" || utf8.RuneCountInString(f.Name) > 100 {
        return in, "Name is required (max 100 characters)"
    }
    roll, err := strconv.Atoi(f.Roll)
    if err != nil || roll <= 0 || roll > 100000 {
        return in, "Roll number must be a positive number"
    }
    classID, err := strconv.Atoi(f.ClassID)
    if err != nil || classID <= 0 {
        return in, "Please choose a class"
    }
    if _, err := models.GetClass(a.DB, classID); err != nil {
        return in, "Please choose a valid class"
    }

    sectionID := 0
    if f.SectionID != "" {
        sectionID, err = strconv.Atoi(f.SectionID)
        if err != nil || sectionID <= 0 {
            return in, "Invalid section"
        }
        ok, err := models.SectionBelongsToClass(a.DB, sectionID, classID)
        if err != nil || !ok {
            return in, "That section does not belong to the chosen class"
        }
    }

    if utf8.RuneCountInString(f.GuardianName) > 100 {
        return in, "Guardian name is too long (max 100 characters)"
    }
    if !validPhone(f.Phone) {
        return in, "Phone may only contain digits, spaces, + and - (max 20 characters)"
    }
    if !editing {
        if msg := validateCredentials(f.Username, f.Password); msg != "" {
            return in, msg
        }
    }

    return models.StudentInput{
        Name:         f.Name,
        Roll:         roll,
        ClassID:      classID,
        SectionID:    sectionID,
        GuardianName: f.GuardianName,
        Phone:        f.Phone,
        Username:     f.Username,
        Password:     f.Password,
    }, ""
}

func (a *App) renderStudentForm(w http.ResponseWriter, r *http.Request, status int, data StudentFormData, msg string) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    sections, err := models.ListAllSections(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data.Classes = classes
    data.Sections = sections

    title := "Add student"
    data.Action = "/admin/students"
    if data.Editing {
        title = "Edit student"
        data.Action = "/admin/students/" + strconv.Itoa(data.ID)
    }
    a.render(w, r, "student_form", status, PageData{Title: title, Error: msg, Data: data})
}

func (a *App) StudentList(w http.ResponseWriter, r *http.Request) {
    students, err := models.ListStudents(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "students", http.StatusOK, PageData{Title: "Students", Data: students})
}

func (a *App) StudentNewForm(w http.ResponseWriter, r *http.Request) {
    a.renderStudentForm(w, r, http.StatusOK, StudentFormData{}, "")
}

func (a *App) StudentCreate(w http.ResponseWriter, r *http.Request) {
    f := readStudentForm(r)
    data := StudentFormData{Form: f}

    in, msg := a.validateStudent(f, false)
    if msg != "" {
        a.renderStudentForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.CreateStudent(a.DB, in); err != nil {
        if models.IsDuplicateOn(err, "users.username") {
            a.renderStudentForm(w, r, http.StatusConflict, data, "That username is already taken")
            return
        }
        if models.IsDuplicate(err) {
            a.renderStudentForm(w, r, http.StatusConflict, data, "That roll number is already used in this class")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students", http.StatusSeeOther)
}

func (a *App) StudentEditForm(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    s, err := models.GetStudent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderStudentForm(w, r, http.StatusOK, StudentFormData{
        Editing:  true,
        ID:       s.ID,
        Username: s.Username,
        Form: StudentForm{
            Name:         s.Name,
            Roll:         strconv.Itoa(s.Roll),
            ClassID:      idString(s.ClassID),
            SectionID:    idString(s.SectionID),
            GuardianName: s.GuardianName,
            Phone:        s.Phone,
        },
    }, "")
}

func (a *App) StudentUpdate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    s, err := models.GetStudent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }

    f := readStudentForm(r)
    data := StudentFormData{Editing: true, ID: s.ID, Username: s.Username, Form: f}

    in, msg := a.validateStudent(f, true)
    if msg != "" {
        a.renderStudentForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.UpdateStudent(a.DB, id, in); err != nil {
        if models.IsDuplicate(err) {
            a.renderStudentForm(w, r, http.StatusConflict, data, "That roll number is already used in this class")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students", http.StatusSeeOther)
}

func (a *App) StudentDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteStudent(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students", http.StatusSeeOther)
}
