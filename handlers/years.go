package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const maxYearName = 30

// workingYearKey is the session value that remembers which year an admin is looking at.
// 0 (not set) means "the default year" and -1 means "all years".
const workingYearKey = "working_year"

// workingYear returns every year and the one the admin is working in (nil = all years, or no
// years exist yet, in which case nothing is filtered).
func (a *App) workingYear(r *http.Request) ([]models.AcademicYear, *models.AcademicYear) {
    years, err := models.ListYears(a.DB)
    if err != nil || len(years) == 0 {
        return nil, nil
    }
    switch chosen := a.Sessions.GetInt(r.Context(), workingYearKey); {
    case chosen == -1:
        return years, nil
    case chosen > 0:
        for i := range years {
            if years[i].ID == chosen {
                return years, &years[i]
            }
        }
    }
    return years, models.DefaultYear(years, today())
}

type YearsPageData struct {
    Years []models.AcademicYear
    Name  string
    Start string
    End   string
}

func (a *App) renderYears(w http.ResponseWriter, r *http.Request, status int, form YearsPageData, msg, notice string) {
    years, err := models.ListYears(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    form.Years = years
    _ = notice
    a.render(w, r, "years", status, PageData{Title: "Academic years", Error: msg, Data: form})
}

func (a *App) YearsPage(w http.ResponseWriter, r *http.Request) {
    a.renderYears(w, r, http.StatusOK, YearsPageData{}, "", "")
}

func (a *App) YearCreate(w http.ResponseWriter, r *http.Request) {
    name := strings.TrimSpace(r.PostFormValue("name"))
    start := r.PostFormValue("start_date")
    end := r.PostFormValue("end_date")
    form := YearsPageData{Name: name, Start: start, End: end}
    fail := func(status int, msg string) { a.renderYears(w, r, status, form, msg, "") }

    if name == "" {
        fail(http.StatusBadRequest, "Please give the year a name, for example 2026")
        return
    }
    if utf8.RuneCountInString(name) > maxYearName {
        fail(http.StatusBadRequest, "The name is too long (max "+strconv.Itoa(maxYearName)+" characters)")
        return
    }
    s, err1 := time.Parse(dateLayout, start)
    e, err2 := time.Parse(dateLayout, end)
    if err1 != nil || err2 != nil {
        fail(http.StatusBadRequest, "Please choose a start date and an end date")
        return
    }
    if !e.After(s) {
        fail(http.StatusBadRequest, "The end date must be after the start date")
        return
    }
    if e.Sub(s).Hours()/24 > 800 {
        fail(http.StatusBadRequest, "A school year cannot be longer than about two years")
        return
    }
    other, err := models.YearOverlaps(a.DB, start, end)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if other != nil {
        fail(http.StatusConflict, "These dates overlap with "+other.Name+" ("+other.StartDate+" to "+other.EndDate+"). Years cannot share days, or an exam could belong to two years.")
        return
    }
    if err := models.CreateYear(a.DB, name, start, end); err != nil {
        if models.IsDuplicate(err) {
            fail(http.StatusConflict, "A year with that name already exists")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/years", http.StatusSeeOther)
}

func (a *App) yearFromPath(w http.ResponseWriter, r *http.Request) (*models.AcademicYear, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        a.NotFound(w, r)
        return nil, false
    }
    y, err := models.GetYear(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        a.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return y, true
}

func (a *App) YearMakeCurrent(w http.ResponseWriter, r *http.Request) {
    y, ok := a.yearFromPath(w, r)
    if !ok {
        return
    }
    if err := models.SetCurrentYear(a.DB, y.ID); err != nil {
        a.serverError(w, err)
        return
    }
    // the admin who made the change also starts working in that year
    a.Sessions.Put(r.Context(), workingYearKey, y.ID)
    http.Redirect(w, r, "/admin/years", http.StatusSeeOther)
}

func (a *App) YearDelete(w http.ResponseWriter, r *http.Request) {
    y, ok := a.yearFromPath(w, r)
    if !ok {
        return
    }
    if err := models.DeleteYear(a.DB, y.ID); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/years", http.StatusSeeOther)
}

// YearSwitch remembers which year the admin wants to look at, then goes back to the page
// they were on.
func (a *App) YearSwitch(w http.ResponseWriter, r *http.Request) {
    choice := r.PostFormValue("year_id")
    switch {
    case choice == "all":
        a.Sessions.Put(r.Context(), workingYearKey, -1)
    default:
        id, err := strconv.Atoi(choice)
        if err != nil || id <= 0 {
            http.Error(w, "Bad request", http.StatusBadRequest)
            return
        }
        if _, err := models.GetYear(a.DB, id); err != nil {
            if errors.Is(err, models.ErrNotFound) {
                a.NotFound(w, r)
                return
            }
            a.serverError(w, err)
            return
        }
        a.Sessions.Put(r.Context(), workingYearKey, id)
    }

    next := r.PostFormValue("next")
    if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
        next = "/dashboard" // only ever go to a page on this site
    }
    http.Redirect(w, r, next, http.StatusSeeOther)
}
