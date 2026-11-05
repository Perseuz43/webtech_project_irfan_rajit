package handlers

import (
    "errors"
    "io"
    "net/http"
    "strconv"
    "strings"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type StudentDetailsData struct {
    Student      *models.Student
    Initial      string
    HasPhoto     bool
    PhotoVersion string
    Notice       string
}

func initialOf(name string) string {
    if runes := []rune(strings.TrimSpace(name)); len(runes) > 0 {
        return strings.ToUpper(string(runes[0]))
    }
    return "?"
}

// loadStudentFromPath finds the student named in the URL or answers 404 itself.
func (a *App) loadStudentFromPath(w http.ResponseWriter, r *http.Request) (*models.Student, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        a.NotFound(w, r)
        return nil, false
    }
    s, err := models.GetStudent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        a.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return s, true
}

func (a *App) renderStudentDetails(w http.ResponseWriter, r *http.Request, status int, s *models.Student, msg, notice string) {
    version, err := models.StudentPhotoVersion(a.DB, s.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "student_details", status, PageData{
        Title: "Photo and details",
        Error: msg,
        Data: StudentDetailsData{
            Student: s, Initial: initialOf(s.Name),
            HasPhoto: version != "", PhotoVersion: version, Notice: notice,
        },
    })
}

func (a *App) StudentDetails(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    notice := ""
    switch r.URL.Query().Get("notice") {
    case "photo":
        notice = "The photo was saved."
    case "photo-removed":
        notice = "The photo was removed."
    }
    a.renderStudentDetails(w, r, http.StatusOK, s, "", notice)
}

func (a *App) StudentPhotoUpload(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    fail := func(msg string) { a.renderStudentDetails(w, r, http.StatusBadRequest, s, msg, "") }

    r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+(64<<10))
    if err := r.ParseMultipartForm(maxPhotoBytes); err != nil {
        fail("The upload failed or the picture is too large (max 3 MB)")
        return
    }
    if r.MultipartForm != nil {
        defer r.MultipartForm.RemoveAll()
    }
    file, _, err := r.FormFile("photo")
    if err != nil {
        fail("Please choose a picture")
        return
    }
    defer file.Close()

    data, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
    if err != nil || len(data) == 0 {
        fail("The file is empty or could not be read")
        return
    }
    if len(data) > maxPhotoBytes {
        fail("The picture is too large (max 3 MB)")
        return
    }

    jpegData, msg := processPhoto(data)
    if msg != "" {
        fail(msg)
        return
    }
    if err := models.SetStudentPhoto(a.DB, s.ID, "image/jpeg", jpegData); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students/"+strconv.Itoa(s.ID)+"/details?notice=photo", http.StatusSeeOther)
}

func (a *App) StudentPhotoRemove(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    if err := models.DeleteStudentPhoto(a.DB, s.ID); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students/"+strconv.Itoa(s.ID)+"/details?notice=photo-removed", http.StatusSeeOther)
}

// canSeeStudent says whether the logged-in user may look at this student's photo and
// details: admins any student, teachers students of their classes, a student only
// themselves, and a parent only their own children.
func (a *App) canSeeStudent(r *http.Request, s *models.Student) (bool, error) {
    u := a.currentUser(r)
    if u == nil {
        return false, nil
    }
    switch u.Role {
    case "admin":
        return true, nil
    case "teacher":
        classes, err := models.ClassesForUser(a.DB, "teacher", u.ID)
        if err != nil {
            return false, err
        }
        for _, c := range classes {
            if c.ID == s.ClassID {
                return true, nil
            }
        }
        return false, nil
    case "student":
        return s.UserID == u.ID, nil
    case "parent":
        p, err := models.GetParentByUserID(a.DB, u.ID)
        if errors.Is(err, models.ErrNotFound) {
            return false, nil
        }
        if err != nil {
            return false, err
        }
        return models.ParentHasChild(a.DB, p.ID, s.ID)
    }
    return false, nil
}

// StudentPhotoServe sends a student's photo to people who are allowed to see it.
func (a *App) StudentPhotoServe(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    allowed, err := a.canSeeStudent(r, s)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if !allowed {
        a.NotFound(w, r)
        return
    }
    contentType, data, err := models.GetStudentPhoto(a.DB, s.ID)
    if errors.Is(err, models.ErrNotFound) {
        a.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    w.Header().Set("Content-Type", contentType)
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
    w.Header().Set("Cache-Control", "private, max-age=86400")
    w.Write(data)
}
