package handlers

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type TTCell struct {
    Day       int
    SubjectID int
    TeacherID int
}

type TTRow struct {
    Period int
    Start  string
    End    string
    Cells  []TTCell // one per day, Sunday first
}

type TimetableEditData struct {
    Classes  []models.Class
    ClassID  int
    Class    *models.Class
    Subjects []models.Subject
    Teachers []models.Teacher
    Days     []string
    Rows     []TTRow
    Notice   string
}

// buildGrid turns the cells (day, period) into rows for the editor. It shows at least
// DefaultPeriods rows, and one empty row more than the highest period in use.
func buildGrid(cells map[[2]int]TTCell, times map[int]models.PeriodTime) []TTRow {
    rowsShown := models.DefaultPeriods
    for key, c := range cells {
        if c.SubjectID > 0 && key[1]+1 > rowsShown {
            rowsShown = key[1] + 1
        }
    }
    if rowsShown > models.MaxPeriods {
        rowsShown = models.MaxPeriods
    }
    grid := make([]TTRow, 0, rowsShown)
    for p := 1; p <= rowsShown; p++ {
        row := TTRow{Period: p, Start: times[p].Start, End: times[p].End}
        for d := range models.DayNames {
            c := cells[[2]int{d, p}]
            c.Day = d
            row.Cells = append(row.Cells, c)
        }
        grid = append(grid, row)
    }
    return grid
}

func (a *App) renderTimetableEditor(w http.ResponseWriter, r *http.Request, status int, classID int, cells map[[2]int]TTCell, msg, notice string) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    d := TimetableEditData{Classes: classes, ClassID: classID, Days: models.DayNames, Notice: notice}

    if classID > 0 {
        class, err := models.GetClass(a.DB, classID)
        if errors.Is(err, models.ErrNotFound) {
            a.NotFound(w, r)
            return
        }
        if err != nil {
            a.serverError(w, err)
            return
        }
        d.Class = class
        if d.Subjects, err = models.ListSubjects(a.DB, classID); err != nil {
            a.serverError(w, err)
            return
        }
        if d.Teachers, err = models.ListTeachers(a.DB); err != nil {
            a.serverError(w, err)
            return
        }
        if cells == nil {
            slots, err := models.ListClassSlots(a.DB, classID)
            if err != nil {
                a.serverError(w, err)
                return
            }
            cells = make(map[[2]int]TTCell)
            for _, s := range slots {
                cells[[2]int{s.Day, s.Period}] = TTCell{Day: s.Day, SubjectID: s.SubjectID, TeacherID: s.TeacherID}
            }
        }
        times, err := models.ListPeriodTimes(a.DB)
        if err != nil {
            a.serverError(w, err)
            return
        }
        d.Rows = buildGrid(cells, times)
    }
    a.render(w, r, "timetable_edit", status, PageData{Title: "Class timetable", Error: msg, Data: d})
}

func (a *App) TimetableEditor(w http.ResponseWriter, r *http.Request) {
    classID, _ := strconv.Atoi(r.URL.Query().Get("class_id"))
    notice := ""
    switch r.URL.Query().Get("saved") {
    case "timetable":
        notice = "The timetable was saved."
    case "times":
        notice = "The period times were saved."
    }
    a.renderTimetableEditor(w, r, http.StatusOK, classID, nil, "", notice)
}

func (a *App) TimetableSave(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    classID, err := strconv.Atoi(r.PostFormValue("class_id"))
    if err != nil || classID <= 0 {
        a.NotFound(w, r)
        return
    }
    if _, err := models.GetClass(a.DB, classID); err != nil {
        if errors.Is(err, models.ErrNotFound) {
            a.NotFound(w, r)
            return
        }
        a.serverError(w, err)
        return
    }
    subjects, err := models.ListSubjects(a.DB, classID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    teachers, err := models.ListTeachers(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    validSubject := make(map[int]bool)
    for _, s := range subjects {
        validSubject[s.ID] = true
    }
    validTeacher := make(map[int]bool)
    for _, t := range teachers {
        validTeacher[t.ID] = true
    }

    cells := make(map[[2]int]TTCell)
    var slots []models.SlotInput
    for p := 1; p <= models.MaxPeriods; p++ {
        for d := range models.DayNames {
            key := fmt.Sprintf("_%d_%d", d, p)
            sid, _ := strconv.Atoi(r.PostFormValue("s" + key))
            tid, _ := strconv.Atoi(r.PostFormValue("t" + key))
            if sid <= 0 {
                continue // an empty lesson (a teacher without a subject is ignored)
            }
            cells[[2]int{d, p}] = TTCell{Day: d, SubjectID: sid, TeacherID: tid}
            if !validSubject[sid] {
                a.renderTimetableEditor(w, r, http.StatusBadRequest, classID, cells,
                    fmt.Sprintf("%s, period %d: that subject does not belong to this class", models.DayNames[d], p), "")
                return
            }
            if tid != 0 && !validTeacher[tid] {
                a.renderTimetableEditor(w, r, http.StatusBadRequest, classID, cells,
                    fmt.Sprintf("%s, period %d: that teacher does not exist", models.DayNames[d], p), "")
                return
            }
            slots = append(slots, models.SlotInput{Day: d, Period: p, SubjectID: sid, TeacherID: tid})
        }
    }

    if err := models.SaveClassTimetable(a.DB, classID, slots); err != nil {
        var conflict *models.ConflictError
        if errors.As(err, &conflict) {
            a.renderTimetableEditor(w, r, http.StatusConflict, classID, cells, conflict.Message+". Nothing was saved.", "")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/timetable?class_id="+strconv.Itoa(classID)+"&saved=timetable", http.StatusSeeOther)
}

// TimetablePeriods saves the start and end time of each period (the same for every class).
func (a *App) TimetablePeriods(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    classID, _ := strconv.Atoi(r.PostFormValue("class_id"))

    var times []models.PeriodTime
    for p := 1; p <= models.MaxPeriods; p++ {
        start := r.PostFormValue("start_" + strconv.Itoa(p))
        end := r.PostFormValue("end_" + strconv.Itoa(p))
        if start == "" && end == "" {
            continue
        }
        msg := ""
        switch {
        case !models.ValidClock(start) || !models.ValidClock(end):
            msg = fmt.Sprintf("Period %d: please give both a start and an end time", p)
        case end <= start:
            msg = fmt.Sprintf("Period %d: the end time must be after the start time", p)
        }
        if msg != "" {
            a.renderTimetableEditor(w, r, http.StatusBadRequest, classID, nil, msg, "")
            return
        }
        times = append(times, models.PeriodTime{Period: p, Start: start, End: end})
    }
    if err := models.SavePeriodTimes(a.DB, times); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/timetable?class_id="+strconv.Itoa(classID)+"&saved=times", http.StatusSeeOther)
}
