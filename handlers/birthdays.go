package handlers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

// startOfToday is midnight today in the server's time zone.
func startOfToday() time.Time {
    n := time.Now()
    return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
}

type BirthdaysData struct {
    Today    []models.BirthdayEntry
    Upcoming []models.BirthdayEntry
    Days     int
    Choices  []BirthdayRange
    NoDOB    int // admin only: students whose date of birth is missing
    IsAdmin  bool
}

type BirthdayRange struct {
    Days  int
    Label string
}

var birthdayRanges = []BirthdayRange{
    {7, "This week"},
    {30, "Next 30 days"},
    {90, "Next 3 months"},
    {366, "Whole year"},
}

// BirthdaysPage lists upcoming student birthdays for admins and teachers.
func (a *App) BirthdaysPage(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)

    days := 30
    if n, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil {
        for _, c := range birthdayRanges {
            if c.Days == n {
                days = n
            }
        }
    }

    list, err := models.ListBirthdays(a.DB, u.Role, u.ID, startOfToday(), days)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data := BirthdaysData{Days: days, Choices: birthdayRanges, IsAdmin: u.Role == "admin"}
    for _, b := range list {
        if b.Today() {
            data.Today = append(data.Today, b)
        } else {
            data.Upcoming = append(data.Upcoming, b)
        }
    }
    if data.IsAdmin {
        if data.NoDOB, err = models.CountStudentsWithoutDOB(a.DB); err != nil {
            a.serverError(w, err)
            return
        }
    }
    a.render(w, r, "birthdays", http.StatusOK, PageData{Title: "Birthdays", Data: data})
}

// StudentDOBSave saves (or clears) a student's date of birth from the details page.
func (a *App) StudentDOBSave(w http.ResponseWriter, r *http.Request) {
    s, ok := a.loadStudentFromPath(w, r)
    if !ok {
        return
    }
    dob := r.PostFormValue("dob")
    if dob != "" && !models.ValidDOB(dob, startOfToday()) {
        a.renderStudentDetails(w, r, http.StatusBadRequest, s,
            "Please enter a real date of birth that is not in the future (for example 2012-05-17)", "")
        return
    }
    if err := models.SetStudentDOB(a.DB, s.ID, dob); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/students/"+strconv.Itoa(s.ID)+"/details?notice=dob", http.StatusSeeOther)
}
