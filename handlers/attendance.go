package handlers

import (
    "errors"
    "net/http"
    "net/url"
    "strconv"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const (
    dateLayout        = "2006-01-02"
    myAttendanceLimit = 100
)

type AttendanceStudent struct {
    ID     int
    Roll   int
    Name   string
    Status string
}

type AttendancePageData struct {
    Classes   []models.Class
    Sections  []models.Section
    ClassID   int
    SectionID int
    Date      string
    Today     string
    Students  []AttendanceStudent
    Loaded    bool
    Saved     bool
}

type ReportPageData struct {
    Classes   []models.Class
    Sections  []models.Section
    ClassID   int
    SectionID int
    From      string
    To        string
    Rows      []models.ClassReportRow
    Loaded    bool
}

type MyAttendanceData struct {
    Student *models.Student
    Summary models.AttendanceSummary
    Records []models.AttendanceRecord
}

type selection struct {
    ClassID   int
    SectionID int
    Date      string
}

func today() string {
    return time.Now().Format(dateLayout)
}

func hasClass(classes []models.Class, id int) bool {
    for _, c := range classes {
        if c.ID == id {
            return true
        }
    }
    return false
}

// classChoices returns the classes this user may use and the sections that belong to them.
func (a *App) classChoices(r *http.Request) ([]models.Class, []models.Section, error) {
    u := a.currentUser(r)
    classes, err := models.ClassesForUser(a.DB, u.Role, u.ID)
    if err != nil {
        return nil, nil, err
    }
    all, err := models.ListAllSections(a.DB)
    if err != nil {
        return nil, nil, err
    }
    var sections []models.Section
    for _, s := range all {
        if hasClass(classes, s.ClassID) {
            sections = append(sections, s)
        }
    }
    return classes, sections, nil
}

// checkClassSection validates the class and section text from a form or URL.
// It returns the IDs and an error message ("" when everything is fine).
func (a *App) checkClassSection(classes []models.Class, classStr, sectionStr string) (int, int, string, error) {
    classID, err := strconv.Atoi(classStr)
    if err != nil || classID <= 0 {
        return 0, 0, "Please choose a class", nil
    }
    if !hasClass(classes, classID) {
        return 0, 0, "You do not have access to that class", nil
    }
    sectionID := 0
    if sectionStr != "" {
        sectionID, err = strconv.Atoi(sectionStr)
        if err != nil || sectionID <= 0 {
            return classID, 0, "Invalid section", nil
        }
        ok, err := models.SectionBelongsToClass(a.DB, sectionID, classID)
        if err != nil {
            return 0, 0, "", err
        }
        if !ok {
            return classID, 0, "That section does not belong to the chosen class", nil
        }
    }
    return classID, sectionID, "", nil
}

// checkSelection validates class, section and date for marking attendance.
func (a *App) checkSelection(r *http.Request, classStr, sectionStr, dateStr string) (selection, string, error) {
    sel := selection{Date: today()}
    dateMsg := ""

    if dateStr == "" {
        dateStr = sel.Date
    }
    t, err := time.Parse(dateLayout, dateStr)
    if err != nil {
        dateMsg = "Invalid date"
    } else if t.Format(dateLayout) > today() {
        dateMsg = "Attendance cannot be marked for a future date"
    } else {
        sel.Date = t.Format(dateLayout)
    }

    classes, _, err := a.classChoices(r)
    if err != nil {
        return sel, "", err
    }
    classID, sectionID, msg, err := a.checkClassSection(classes, classStr, sectionStr)
    if err != nil {
        return sel, "", err
    }
    sel.ClassID, sel.SectionID = classID, sectionID

    if msg == "" {
        msg = dateMsg
    }
    return sel, msg, nil
}

func (a *App) renderSheet(w http.ResponseWriter, r *http.Request, status int, sel selection, msg string, saved bool) {
    classes, sections, err := a.classChoices(r)
    if err != nil {
        a.serverError(w, err)
        return
    }
    d := AttendancePageData{
        Classes:   classes,
        Sections:  sections,
        ClassID:   sel.ClassID,
        SectionID: sel.SectionID,
        Date:      sel.Date,
        Today:     today(),
        Saved:     saved,
    }

    if msg == "" && sel.ClassID != 0 {
        students, err := models.ListClassStudents(a.DB, sel.ClassID, sel.SectionID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        marks, err := models.GetAttendanceMap(a.DB, sel.ClassID, sel.SectionID, sel.Date)
        if err != nil {
            a.serverError(w, err)
            return
        }
        for _, s := range students {
            st := marks[s.ID]
            if st == "" {
                st = models.StatusPresent // default for students not marked yet
            }
            d.Students = append(d.Students, AttendanceStudent{ID: s.ID, Roll: s.Roll, Name: s.Name, Status: st})
        }
        d.Loaded = true
    }

    a.render(w, r, "attendance", status, PageData{Title: "Take attendance", Error: msg, Data: d})
}

// AttendancePage shows the class picker and, once a class is chosen, the student list.
func (a *App) AttendancePage(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    if q.Get("class_id") == "" {
        a.renderSheet(w, r, http.StatusOK, selection{Date: today()}, "", false)
        return
    }
    sel, msg, err := a.checkSelection(r, q.Get("class_id"), q.Get("section_id"), q.Get("date"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    status := http.StatusOK
    if msg != "" {
        status = http.StatusBadRequest
    }
    a.renderSheet(w, r, status, sel, msg, q.Get("saved") == "1")
}

// AttendanceSave stores the marks for every student in the chosen class and section.
func (a *App) AttendanceSave(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    sel, msg, err := a.checkSelection(r, r.PostFormValue("class_id"), r.PostFormValue("section_id"), r.PostFormValue("date"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    if msg != "" {
        a.renderSheet(w, r, http.StatusBadRequest, sel, msg, false)
        return
    }

    // only students that really belong to this class and section are saved
    students, err := models.ListClassStudents(a.DB, sel.ClassID, sel.SectionID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if len(students) == 0 {
        a.renderSheet(w, r, http.StatusBadRequest, sel, "There are no students to mark", false)
        return
    }
    marks := make(map[int]string, len(students))
    for _, s := range students {
        st := r.PostFormValue("status_" + strconv.Itoa(s.ID))
        if !models.ValidStatus(st) {
            a.renderSheet(w, r, http.StatusBadRequest, sel, "Please mark every student before saving", false)
            return
        }
        marks[s.ID] = st
    }

    if err := models.SaveAttendance(a.DB, sel.Date, marks); err != nil {
        a.serverError(w, err)
        return
    }

    v := url.Values{}
    v.Set("class_id", strconv.Itoa(sel.ClassID))
    if sel.SectionID != 0 {
        v.Set("section_id", strconv.Itoa(sel.SectionID))
    }
    v.Set("date", sel.Date)
    v.Set("saved", "1")
    http.Redirect(w, r, "/attendance?"+v.Encode(), http.StatusSeeOther)
}

// AttendanceReport shows each student's attendance for a class between two dates.
func (a *App) AttendanceReport(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    classes, sections, err := a.classChoices(r)
    if err != nil {
        a.serverError(w, err)
        return
    }
    d := ReportPageData{
        Classes:  classes,
        Sections: sections,
        From:     time.Now().Format("2006-01") + "-01",
        To:       today(),
    }
    if q.Get("class_id") == "" {
        a.render(w, r, "attendance_report", http.StatusOK, PageData{Title: "Attendance report", Data: d})
        return
    }

    msg := ""
    from, err1 := time.Parse(dateLayout, q.Get("from"))
    to, err2 := time.Parse(dateLayout, q.Get("to"))
    if err1 != nil || err2 != nil {
        msg = "Please enter valid dates"
    } else if from.After(to) {
        msg = "The start date must not be after the end date"
    } else {
        d.From, d.To = from.Format(dateLayout), to.Format(dateLayout)
    }

    classID, sectionID, classMsg, err := a.checkClassSection(classes, q.Get("class_id"), q.Get("section_id"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    d.ClassID, d.SectionID = classID, sectionID
    if classMsg != "" {
        msg = classMsg
    }
    if msg != "" {
        a.render(w, r, "attendance_report", http.StatusBadRequest, PageData{Title: "Attendance report", Error: msg, Data: d})
        return
    }

    rows, err := models.ClassReport(a.DB, classID, sectionID, d.From, d.To)
    if err != nil {
        a.serverError(w, err)
        return
    }
    d.Rows = rows
    d.Loaded = true
    a.render(w, r, "attendance_report", http.StatusOK, PageData{Title: "Attendance report", Data: d})
}

// MyAttendance shows the logged-in student's own attendance.
func (a *App) MyAttendance(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    s, err := models.GetStudentByUserID(a.DB, u.ID)
    if errors.Is(err, models.ErrNotFound) {
        a.render(w, r, "my_attendance", http.StatusOK, PageData{Title: "My attendance", Error: "No student profile was found for your account"})
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
    a.render(w, r, "my_attendance", http.StatusOK, PageData{
        Title: "My attendance",
        Data:  MyAttendanceData{Student: s, Summary: summary, Records: records},
    })
}
