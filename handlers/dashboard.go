package handlers

import (
    "errors"
    "net/http"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type DashboardData struct {
    Student  *models.Student
    Teacher  *models.Teacher
    Subjects []models.SubjectOption
}

// Dashboard shows different content depending on the logged-in user's role.
func (a *App) Dashboard(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    if u == nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    var data DashboardData
    switch u.Role {
    case "student":
        s, err := models.GetStudentByUserID(a.DB, u.ID)
        if err != nil && !errors.Is(err, models.ErrNotFound) {
            a.serverError(w, err)
            return
        }
        data.Student = s
    case "teacher":
        t, err := models.GetTeacherByUserID(a.DB, u.ID)
        if err != nil && !errors.Is(err, models.ErrNotFound) {
            a.serverError(w, err)
            return
        }
        data.Teacher = t
        if t != nil {
            subjects, err := models.ListAssignedSubjects(a.DB, t.ID)
            if err != nil {
                a.serverError(w, err)
                return
            }
            data.Subjects = subjects
        }
    }
    a.render(w, r, "dashboard", http.StatusOK, PageData{Title: "Dashboard", Data: data})
}
