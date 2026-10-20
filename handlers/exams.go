package handlers

import (
    "errors"
    "math"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type ExamListData struct {
    Exams   []models.Exam
    Classes []models.Class
    Today   string
}

type ExamDetailData struct {
    Exam     *models.Exam
    Subjects []models.ExamSubject
    Cards    []*models.ReportCard
    Scale    *models.GradingScale
}

type ReportCardPage struct {
    Card *models.ReportCard
    Back string
}

func (a *App) renderExamList(w http.ResponseWriter, r *http.Request, status int, msg string) {
    exams, err := models.ListExams(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "exams", status, PageData{
        Title: "Exams",
        Error: msg,
        Data:  ExamListData{Exams: exams, Classes: classes, Today: today()},
    })
}

func (a *App) ExamList(w http.ResponseWriter, r *http.Request) {
    a.renderExamList(w, r, http.StatusOK, "")
}

func (a *App) ExamCreate(w http.ResponseWriter, r *http.Request) {
    name, msg := cleanName(r)
    classID, classErr := strconv.Atoi(r.PostFormValue("class_id"))
    date := strings.TrimSpace(r.PostFormValue("exam_date"))

    if msg == "" {
        if classErr != nil || classID <= 0 {
            msg = "Please choose a class"
        } else if _, err := time.Parse(dateLayout, date); err != nil {
            msg = "Please enter a valid date"
        } else if _, err := models.GetClass(a.DB, classID); err != nil {
            msg = "Please choose a valid class"
        }
    }
    if msg != "" {
        a.renderExamList(w, r, http.StatusBadRequest, msg)
        return
    }

    id, err := models.CreateExam(a.DB, classID, name, date)
    if err != nil {
        if models.IsDuplicate(err) {
            a.renderExamList(w, r, http.StatusConflict, "That class already has an exam with this name")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/exams/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) renderExamDetail(w http.ResponseWriter, r *http.Request, id, status int, msg string) {
    exam, err := models.GetExam(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    if err := models.SyncExamSubjects(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    subjects, err := models.ListExamSubjects(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    students, err := models.ListClassStudents(a.DB, exam.ClassID, 0)
    if err != nil {
        a.serverError(w, err)
        return
    }
    scale, err := models.ActiveScale(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }

    cards := make([]*models.ReportCard, 0, len(students))
    for i := range students {
        c, err := models.BuildReportCard(a.DB, &students[i], exam, scale)
        if err != nil {
            a.serverError(w, err)
            return
        }
        cards = append(cards, c)
    }

    a.render(w, r, "exam_detail", status, PageData{
        Title: exam.Name,
        Error: msg,
        Data:  ExamDetailData{Exam: exam, Subjects: subjects, Cards: cards, Scale: scale},
    })
}

func (a *App) ExamDetail(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    a.renderExamDetail(w, r, id, http.StatusOK, "")
}

func (a *App) ExamSaveSubjects(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    subjects, err := models.ListExamSubjects(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }

    full := make(map[int]float64, len(subjects))
    for _, s := range subjects {
        raw := strings.TrimSpace(r.PostFormValue("full_" + strconv.Itoa(s.SubjectID)))
        v, err := strconv.ParseFloat(raw, 64)
        if err != nil || math.IsNaN(v) || v <= 0 || v > 1000 {
            a.renderExamDetail(w, r, id, http.StatusBadRequest,
                "Full marks for "+s.Name+" must be a number above 0 and up to 1000")
            return
        }
        full[s.SubjectID] = v
    }

    conflict, err := models.SaveFullMarks(a.DB, id, full)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if conflict != "" {
        a.renderExamDetail(w, r, id, http.StatusConflict,
            "Full marks for "+conflict+" cannot be lower than marks that are already entered")
        return
    }
    http.Redirect(w, r, "/admin/exams/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) ExamPublish(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.SetExamPublished(a.DB, id, r.PostFormValue("publish") == "1"); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/exams/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) ExamDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteExam(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/exams", http.StatusSeeOther)
}

// renderCard builds and shows one student's report card using the active grading system.
func (a *App) renderCard(w http.ResponseWriter, r *http.Request, student *models.Student, exam *models.Exam, back string) {
    if err := models.SyncExamSubjects(a.DB, exam.ID); err != nil {
        a.serverError(w, err)
        return
    }
    scale, err := models.ActiveScale(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    card, err := models.BuildReportCard(a.DB, student, exam, scale)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "report_card", http.StatusOK, PageData{
        Title: "Report card",
        Data:  ReportCardPage{Card: card, Back: back},
    })
}

// ExamStudentCard is the admin's view of any student's report card, published or not.
func (a *App) ExamStudentCard(w http.ResponseWriter, r *http.Request) {
    id, ok1 := pathInt(r, "id")
    sid, ok2 := pathInt(r, "sid")
    if !ok1 || !ok2 {
        http.NotFound(w, r)
        return
    }
    exam, err := models.GetExam(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    student, err := models.GetStudent(a.DB, sid)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    if student.ClassID != exam.ClassID {
        http.NotFound(w, r)
        return
    }
    a.renderCard(w, r, student, exam, "/admin/exams/"+strconv.Itoa(id))
}
