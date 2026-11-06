package handlers

import (
    "errors"
    "net/http"
    "sort"
    "strings"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type TTViewDay struct {
    Index int
    Name  string
    Today bool
}

type TTViewCell struct {
    Subject string
    Detail  string // the teacher (class view) or the class (teacher view)
}

type TTViewRow struct {
    Period int
    Start  string
    End    string
    Cells  []TTViewCell // one per day in Days
}

type TimetableViewData struct {
    Heading string
    Days    []TTViewDay
    Rows    []TTViewRow
    Back    string
}

// buildView arranges lessons into a week grid. Only days and periods that have a lesson are
// shown, so a school that works Sunday to Thursday does not see two empty columns.
// showClass puts the class name under the subject (for teachers) instead of the teacher's name.
func buildView(slots []models.Slot, times map[int]models.PeriodTime, showClass bool) ([]TTViewDay, []TTViewRow) {
    dayUsed := make(map[int]bool)
    periodUsed := make(map[int]bool)
    cell := make(map[[2]int]TTViewCell)
    for _, s := range slots {
        dayUsed[s.Day] = true
        periodUsed[s.Period] = true
        detail := s.TeacherName
        if showClass {
            detail = s.ClassName
        }
        key := [2]int{s.Day, s.Period}
        if old, ok := cell[key]; ok { // should not happen, but never hide a lesson
            cell[key] = TTViewCell{Subject: old.Subject + " / " + s.SubjectName, Detail: strings.TrimSpace(old.Detail + " " + detail)}
        } else {
            cell[key] = TTViewCell{Subject: s.SubjectName, Detail: detail}
        }
    }

    today := int(time.Now().Weekday())
    var days []TTViewDay
    for d := range models.DayNames {
        if dayUsed[d] {
            days = append(days, TTViewDay{Index: d, Name: models.DayNames[d], Today: d == today})
        }
    }
    var periods []int
    for p := range periodUsed {
        periods = append(periods, p)
    }
    sort.Ints(periods)

    var rows []TTViewRow
    for _, p := range periods {
        row := TTViewRow{Period: p, Start: times[p].Start, End: times[p].End}
        for _, d := range days {
            row.Cells = append(row.Cells, cell[[2]int{d.Index, p}])
        }
        rows = append(rows, row)
    }
    return days, rows
}

func (a *App) renderTimetableView(w http.ResponseWriter, r *http.Request, heading string, slots []models.Slot, showClass bool, back string) {
    times, err := models.ListPeriodTimes(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    days, rows := buildView(slots, times, showClass)
    a.render(w, r, "timetable_view", http.StatusOK, PageData{
        Title: "Timetable",
        Data:  TimetableViewData{Heading: heading, Days: days, Rows: rows, Back: back},
    })
}

// Timetable shows the logged-in person's own week: a student sees their class, a teacher sees
// the lessons they give. Admins use the editor, and parents open a child's timetable.
func (a *App) Timetable(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    switch u.Role {
    case "admin":
        http.Redirect(w, r, "/admin/timetable", http.StatusSeeOther)
    case "parent":
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
    case "student":
        s, ok := a.currentStudent(w, r)
        if !ok {
            return
        }
        a.studentTimetable(w, r, s, "My timetable", "/dashboard")
    case "teacher":
        t, err := models.GetTeacherByUserID(a.DB, u.ID)
        if errors.Is(err, models.ErrNotFound) {
            a.NotFound(w, r)
            return
        }
        if err != nil {
            a.serverError(w, err)
            return
        }
        slots, err := models.ListTeacherSlots(a.DB, t.ID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        a.renderTimetableView(w, r, "My timetable", slots, true, "/dashboard")
    default:
        a.NotFound(w, r)
    }
}

func (a *App) studentTimetable(w http.ResponseWriter, r *http.Request, s *models.Student, heading, back string) {
    var slots []models.Slot
    if s.ClassID != 0 {
        var err error
        if slots, err = models.ListClassSlots(a.DB, s.ClassID); err != nil {
            a.serverError(w, err)
            return
        }
    }
    a.renderTimetableView(w, r, heading, slots, false, back)
}

// ChildTimetable shows a linked child's timetable to their parent.
func (a *App) ChildTimetable(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        a.studentTimetable(w, r, s, s.Name+": timetable", "/dashboard")
    }
}

// lessonsToday fills the dashboard's "Today's lessons" for a student or a teacher.
func (a *App) lessonsToday(u *SessionUser, data *DashboardData) error {
    day := int(time.Now().Weekday())
    var err error
    switch {
    case u.Role == "student" && data.Student != nil && data.Student.ClassID != 0:
        data.TodayLessons, err = models.ListClassSlotsOnDay(a.DB, data.Student.ClassID, day)
    case u.Role == "teacher" && data.Teacher != nil:
        data.TodayLessons, err = models.ListTeacherSlotsOnDay(a.DB, data.Teacher.ID, day)
    default:
        return nil
    }
    if err != nil || len(data.TodayLessons) == 0 {
        return err
    }
    data.PeriodTimes, err = models.ListPeriodTimes(a.DB)
    return err
}

