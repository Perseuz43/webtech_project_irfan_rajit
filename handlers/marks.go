package handlers

import (
    "fmt"
    "math"
    "net/http"
    "strconv"
    "strings"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type MarkRow struct {
    StudentID int
    Roll      int
    Name      string
    Value     string
}

type MarksPageData struct {
    Exams     []models.Exam
    ExamID    int
    Exam      *models.Exam
    Subjects  []models.ExamSubject
    SubjectID int
    Subject   *models.ExamSubject
    Rows      []MarkRow
    Loaded    bool
    Saved     bool
}

type markCtx struct {
    exams    []models.Exam
    exam     *models.Exam
    subjects []models.ExamSubject
    subject  *models.ExamSubject
}

// markContext checks that the user may use the chosen exam and subject.
// It returns an error message ("" when everything is fine).
func (a *App) markContext(r *http.Request, examStr, subjectStr string) (*markCtx, string, error) {
    u := a.currentUser(r)
    exams, err := models.ListExamsForUser(a.DB, u.Role, u.ID)
    if err != nil {
        return nil, "", err
    }
    mc := &markCtx{exams: exams}
    if examStr == "" {
        return mc, "", nil
    }

    examID, err := strconv.Atoi(examStr)
    if err != nil || examID <= 0 {
        return mc, "Invalid exam", nil
    }
    for i := range exams {
        if exams[i].ID == examID {
            mc.exam = &exams[i]
        }
    }
    if mc.exam == nil {
        return mc, "You do not have access to that exam", nil
    }

    if err := models.SyncExamSubjects(a.DB, examID); err != nil {
        return nil, "", err
    }
    mc.subjects, err = models.ListMarkableSubjects(a.DB, u.Role, u.ID, examID)
    if err != nil {
        return nil, "", err
    }
    if subjectStr == "" {
        return mc, "", nil
    }

    subjectID, err := strconv.Atoi(subjectStr)
    if err != nil || subjectID <= 0 {
        return mc, "Invalid subject", nil
    }
    for i := range mc.subjects {
        if mc.subjects[i].SubjectID == subjectID {
            mc.subject = &mc.subjects[i]
        }
    }
    if mc.subject == nil {
        return mc, "You cannot enter marks for that subject", nil
    }
    return mc, "", nil
}

func (a *App) renderMarks(w http.ResponseWriter, r *http.Request, status int, mc *markCtx, msg string, saved bool, posted map[int]string) {
    d := MarksPageData{Exams: mc.exams, Exam: mc.exam, Subjects: mc.subjects, Subject: mc.subject, Saved: saved}
    if mc.exam != nil {
        d.ExamID = mc.exam.ID
    }
    if mc.subject != nil {
        d.SubjectID = mc.subject.SubjectID

        students, err := models.ListClassStudents(a.DB, mc.exam.ClassID, 0)
        if err != nil {
            a.serverError(w, err)
            return
        }
        existing, err := models.GetMarksMap(a.DB, mc.exam.ID, mc.subject.SubjectID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        for _, s := range students {
            val := ""
            if posted != nil {
                val = posted[s.ID]
            } else if v, ok := existing[s.ID]; ok {
                val = fmt.Sprintf("%g", v)
            }
            d.Rows = append(d.Rows, MarkRow{StudentID: s.ID, Roll: s.Roll, Name: s.Name, Value: val})
        }
        d.Loaded = true
    }
    a.render(w, r, "marks", status, PageData{Title: "Enter marks", Error: msg, Data: d})
}

func (a *App) MarksPage(w http.ResponseWriter, r *http.Request) {
    q := r.URL.Query()
    mc, msg, err := a.markContext(r, q.Get("exam_id"), q.Get("subject_id"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    status := http.StatusOK
    if msg != "" {
        status = http.StatusBadRequest
    }
    a.renderMarks(w, r, status, mc, msg, q.Get("saved") == "1", nil)
}

func (a *App) MarksSave(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    mc, msg, err := a.markContext(r, r.PostFormValue("exam_id"), r.PostFormValue("subject_id"))
    if err != nil {
        a.serverError(w, err)
        return
    }
    if msg == "" && mc.subject == nil {
        msg = "Please choose an exam and a subject"
    }
    if msg != "" {
        a.renderMarks(w, r, http.StatusBadRequest, mc, msg, false, nil)
        return
    }

    students, err := models.ListClassStudents(a.DB, mc.exam.ClassID, 0)
    if err != nil {
        a.serverError(w, err)
        return
    }

    entries := make([]models.MarkEntry, 0, len(students))
    posted := make(map[int]string, len(students))
    errMsg := ""
    for _, s := range students {
        raw := strings.TrimSpace(r.PostFormValue("mark_" + strconv.Itoa(s.ID)))
        posted[s.ID] = raw
        if raw == "" {
            entries = append(entries, models.MarkEntry{StudentID: s.ID})
            continue
        }
        v, err := strconv.ParseFloat(raw, 64)
        if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > mc.subject.FullMarks {
            if errMsg == "" {
                errMsg = fmt.Sprintf("%s: marks must be a number from 0 to %g", s.Name, mc.subject.FullMarks)
            }
            continue
        }
        entries = append(entries, models.MarkEntry{StudentID: s.ID, Marks: v, Has: true})
    }
    if errMsg != "" {
        a.renderMarks(w, r, http.StatusBadRequest, mc, errMsg, false, posted)
        return
    }

    if err := models.SaveMarks(a.DB, mc.exam.ID, mc.subject.SubjectID, entries); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r,
        fmt.Sprintf("/marks?exam_id=%d&subject_id=%d&saved=1", mc.exam.ID, mc.subject.SubjectID),
        http.StatusSeeOther)
}
