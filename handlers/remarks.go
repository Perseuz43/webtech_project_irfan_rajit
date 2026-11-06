package handlers

import (
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type RemarkRow struct {
    StudentID int
    Roll      int
    Name      string
    Value     string
}

type RemarksPageData struct {
    Exams     []models.Exam
    ExamID    int
    Exam      *models.Exam
    Rows      []RemarkRow
    Saved     bool
    MaxLength int
}

// remarkExam finds the chosen exam among the exams this user may use.
// The message is "" when all is fine, and the exam is nil when none is chosen.
func (a *App) remarkExam(r *http.Request, examStr string) ([]models.Exam, *models.Exam, string, error) {
    u := a.currentUser(r)
    exams, err := models.ListExamsForUser(a.DB, u.Role, u.ID)
    if err != nil || examStr == "" {
        return exams, nil, "", err
    }
    id, convErr := strconv.Atoi(examStr)
    if convErr != nil || id <= 0 {
        return exams, nil, "Invalid exam", nil
    }
    for i := range exams {
        if exams[i].ID == id {
            return exams, &exams[i], "", nil
        }
    }
    return exams, nil, "You do not have access to that exam", nil
}

func (a *App) renderRemarks(w http.ResponseWriter, r *http.Request, status int, exams []models.Exam, exam *models.Exam, msg string, saved bool, posted map[int]string) {
    d := RemarksPageData{Exams: exams, Exam: exam, Saved: saved, MaxLength: models.MaxRemarkLength}
    if exam != nil {
        d.ExamID = exam.ID
        students, err := models.ListClassStudents(a.DB, exam.ClassID, 0)
        if err != nil {
            a.serverError(w, err)
            return
        }
        existing, err := models.GetRemarksMap(a.DB, exam.ID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        for _, s := range students {
            val := existing[s.ID]
            if posted != nil {
                val = posted[s.ID]
            }
            d.Rows = append(d.Rows, RemarkRow{StudentID: s.ID, Roll: s.Roll, Name: s.Name, Value: val})
        }
    }
    a.render(w, r, "remarks", status, PageData{Title: "Report card remarks", Error: msg, Data: d})
}

func (a *App) RemarksPage(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    exams, exam, msg, err := a.remarkExam(r, q.Get("exam_id"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    status := http.StatusOK
    if msg != "" {
        status = http.StatusBadRequest
    }
    a.renderRemarks(w, r, status, exams, exam, msg, q.Get("saved") == "1", nil)
}

func (a *App) RemarksSave(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    exams, exam, msg, err := a.remarkExam(r, r.PostFormValue("exam_id"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    if exam == nil || msg != "" {
        if msg == "" {
            msg = "Please choose an exam"
        }
        a.renderRemarks(w, r, http.StatusBadRequest, exams, nil, msg, false, nil)
        return
    }

    students, err := models.ListClassStudents(a.DB, exam.ClassID, 0)
    if err != nil {
        a.serverError(w, err)
        return
    }
    posted := make(map[int]string)
    for _, s := range students {
        text := strings.Join(strings.Fields(r.PostFormValue("remark_"+strconv.Itoa(s.ID))), " ")
        if utf8.RuneCountInString(text) > models.MaxRemarkLength {
            posted[s.ID] = text
            a.renderRemarks(w, r, http.StatusBadRequest, exams, exam,
                s.Name+": the remark is too long (max "+strconv.Itoa(models.MaxRemarkLength)+" characters)", false, collectRemarks(students, r))
            return
        }
        posted[s.ID] = text
    }
    if err := models.SaveRemarks(a.DB, exam.ID, posted); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/remarks?exam_id="+strconv.Itoa(exam.ID)+"&saved=1", http.StatusSeeOther)
}

// collectRemarks reads every remark box again, so a failed save keeps what was typed.
func collectRemarks(students []models.Student, r *http.Request) map[int]string {
    m := make(map[int]string, len(students))
    for _, s := range students {
        m[s.ID] = strings.TrimSpace(r.PostFormValue("remark_" + strconv.Itoa(s.ID)))
    }
    return m
}
