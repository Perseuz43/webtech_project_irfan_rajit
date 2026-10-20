package handlers

import (
    "errors"
    "net/http"
    "strconv"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type ResultsListData struct {
    Heading string
    Exams   []models.Exam
    Base    string
}

// currentStudent loads the logged-in student's profile. It answers the request itself when it cannot.
func (a *App) currentStudent(w http.ResponseWriter, r *http.Request) (*models.Student, bool) {
    u := a.currentUser(r)
    s, err := models.GetStudentByUserID(a.DB, u.ID)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return s, true
}

// linkedChild loads the child named in the URL, but only if it is linked to the logged-in parent.
func (a *App) linkedChild(w http.ResponseWriter, r *http.Request) (*models.Student, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return nil, false
    }
    u := a.currentUser(r)
    p, err := models.GetParentByUserID(a.DB, u.ID)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    linked, err := models.ParentHasChild(a.DB, p.ID, id)
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    if !linked {
        http.NotFound(w, r)
        return nil, false
    }
    s, err := models.GetStudent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return s, true
}

func (a *App) resultsList(w http.ResponseWriter, r *http.Request, s *models.Student, heading, base string) {
    var exams []models.Exam
    if s.ClassID != 0 {
        var err error
        exams, err = models.ListPublishedExamsForClass(a.DB, s.ClassID)
        if err != nil {
            a.serverError(w, err)
            return
        }
    }
    a.render(w, r, "results_list", http.StatusOK, PageData{
        Title: "Results",
        Data:  ResultsListData{Heading: heading, Exams: exams, Base: base},
    })
}

// publishedCard shows a report card only for a published exam of the student's own class.
func (a *App) publishedCard(w http.ResponseWriter, r *http.Request, s *models.Student, back string) {
    eid, ok := pathInt(r, "eid")
    if !ok {
        http.NotFound(w, r)
        return
    }
    exam, err := models.GetExam(a.DB, eid)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    if !exam.Published || exam.ClassID != s.ClassID {
        http.NotFound(w, r)
        return
    }
    a.renderCard(w, r, s, exam, back)
}

func (a *App) MyResults(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.currentStudent(w, r); ok {
        a.resultsList(w, r, s, "My results", "/my-results")
    }
}

func (a *App) MyReportCard(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.currentStudent(w, r); ok {
        a.publishedCard(w, r, s, "/my-results")
    }
}

func (a *App) ChildResults(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        a.resultsList(w, r, s, s.Name+": results", "/children/"+strconv.Itoa(s.ID)+"/results")
    }
}

func (a *App) ChildReportCard(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        a.publishedCard(w, r, s, "/children/"+strconv.Itoa(s.ID)+"/results")
    }
}
