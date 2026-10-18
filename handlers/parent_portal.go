package handlers

import (
    "errors"
    "net/http"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type ChildAttendanceData struct {
    Student *models.Student
    Summary models.AttendanceSummary
    Records []models.AttendanceRecord
}

// ChildAttendance shows one child's attendance, but only to a parent linked to that child.
func (a *App) ChildAttendance(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    u := a.currentUser(r)

    p, err := models.GetParentByUserID(a.DB, u.ID)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }

    // a child that is not linked to this parent looks the same as one that does not exist
    linked, err := models.ParentHasChild(a.DB, p.ID, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if !linked {
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
    summary, err := models.GetStudentSummary(a.DB, s.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    records, err := models.GetStudentAttendance(a.DB, s.ID, myAttendanceLimit)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "child_attendance", http.StatusOK, PageData{
        Title: s.Name,
        Data:  ChildAttendanceData{Student: s, Summary: summary, Records: records},
    })
}
