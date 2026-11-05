package handlers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type IDCardsData struct {
    Students  []models.Student
    Classes   []models.Class
    Sections  []models.Section
    ClassID   int
    SectionID int
    Single    bool   // one student, opened from the student list
    Year      string // printed on the card
    NoPhoto   int    // how many of the shown students have no photo yet
}

func (a *App) renderIDCards(w http.ResponseWriter, r *http.Request, data IDCardsData) {
    data.Year = time.Now().Format("2006")
    for _, s := range data.Students {
        if s.PhotoVersion == "" {
            data.NoPhoto++
        }
    }
    a.render(w, r, "id_cards", http.StatusOK, PageData{Title: "ID cards", Data: data})
}

// IDCardOne shows the ID card of a single student.
func (a *App) IDCardOne(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    a.renderIDCards(w, r, IDCardsData{Students: []models.Student{*s}, Single: true})
}

// IDCardsPage shows the cards of a whole class or section (or every student) ready to print.
func (a *App) IDCardsPage(w http.ResponseWriter, r *http.Request) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    sections, err := models.ListAllSections(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data := IDCardsData{Classes: classes, Sections: sections}

    q := r.URL.Query()
    if n, err := strconv.Atoi(q.Get("class_id")); err == nil && n > 0 {
        for _, c := range classes {
            if c.ID == n {
                data.ClassID = n
            }
        }
    }
    if n, err := strconv.Atoi(q.Get("section_id")); err == nil && n > 0 && data.ClassID > 0 {
        if ok, err := models.SectionBelongsToClass(a.DB, n, data.ClassID); err == nil && ok {
            data.SectionID = n
        }
    }

    if data.ClassID > 0 {
        data.Students, err = models.ListClassStudents(a.DB, data.ClassID, data.SectionID)
    } else {
        data.Students, err = models.ListStudents(a.DB)
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderIDCards(w, r, data)
}
