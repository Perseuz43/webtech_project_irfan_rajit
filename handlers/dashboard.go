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
    Parent   *models.Parent
    Children []models.Student
    Notices  []models.Notice

    Birthdays      []models.BirthdayEntry // admins and teachers: the coming week
    MyBirthday     *models.BirthdayEntry  // a student whose birthday is today
    ChildBirthdays []models.BirthdayEntry // a parent's children whose birthday is today

    TodayLessons []models.Slot              // a student's or teacher's lessons today
    PeriodTimes  map[int]models.PeriodTime // clock times for those periods
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
    case "parent":
        p, err := models.GetParentByUserID(a.DB, u.ID)
        if err != nil && !errors.Is(err, models.ErrNotFound) {
            a.serverError(w, err)
            return
        }
        data.Parent = p
        if p != nil {
            children, err := models.ListChildren(a.DB, p.ID)
            if err != nil {
                a.serverError(w, err)
                return
            }
            data.Children = children
        }
    }

    if err := a.lessonsToday(u, &data); err != nil {
        a.serverError(w, err)
        return
    }
    if err := a.loadBirthdays(u, &data); err != nil {
        a.serverError(w, err)
        return
    }

    notices, err := models.ListNoticesForUser(a.DB, u.Role, u.ID, 3)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data.Notices = notices

    a.render(w, r, "dashboard", http.StatusOK, PageData{Title: "Dashboard", Data: data})
}

// loadBirthdays fills in the birthday parts of the dashboard for the logged-in role.
func (a *App) loadBirthdays(u *SessionUser, data *DashboardData) error {
    today := startOfToday()
    switch u.Role {
    case "admin", "teacher":
        list, err := models.ListBirthdays(a.DB, u.Role, u.ID, today, 7)
        if err != nil {
            return err
        }
        data.Birthdays = list
    case "student":
        if data.Student == nil {
            return nil
        }
        b, err := models.StudentBirthday(a.DB, data.Student.ID, today)
        if err != nil {
            return err
        }
        if b != nil && b.Today() {
            data.MyBirthday = b
        }
    case "parent":
        for _, child := range data.Children {
            b, err := models.StudentBirthday(a.DB, child.ID, today)
            if err != nil {
                return err
            }
            if b != nil && b.Today() {
                data.ChildBirthdays = append(data.ChildBirthdays, *b)
            }
        }
    }
    return nil
}
