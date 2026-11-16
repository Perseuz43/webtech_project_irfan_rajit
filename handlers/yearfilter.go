package handlers

import (
    "net/http"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

// inWorkingYear says whether a dated item belongs to the year the admin is viewing.
// With no working year (all years, or none defined) everything is shown.
func inWorkingYear(y *models.AcademicYear, date string) bool {
    return y == nil || y.Contains(date)
}

func (a *App) examsInYear(r *http.Request, list []models.Exam) []models.Exam {
    _, y := a.workingYear(r)
    if y == nil {
        return list
    }
    var out []models.Exam
    for _, e := range list {
        if y.Contains(e.Date) {
            out = append(out, e)
        }
    }
    return out
}

func (a *App) feesInYear(r *http.Request, list []models.FeeRow) []models.FeeRow {
    _, y := a.workingYear(r)
    if y == nil {
        return list
    }
    var out []models.FeeRow
    for _, f := range list {
        if y.Contains(f.DueDate) {
            out = append(out, f)
        }
    }
    return out
}

func (a *App) duesInYear(r *http.Request, list []models.ChargeRow) []models.ChargeRow {
    _, y := a.workingYear(r)
    if y == nil {
        return list
    }
    var out []models.ChargeRow
    for _, c := range list {
        if y.Contains(c.DueDate) {
            out = append(out, c)
        }
    }
    return out
}

// yearDates gives the default From and To dates of a report for the working year: the whole
// year, but not beyond today for a year that is still running.
func yearDates(y *models.AcademicYear, todayDate string) (from, to string) {
    from, to = y.StartDate, y.EndDate
    if todayDate >= y.StartDate && todayDate < y.EndDate {
        to = todayDate
    }
    return
}
