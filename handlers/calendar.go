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

const maxCellEvents = 3 // events drawn inside one day box; the rest are summarised

type CalendarDay struct {
    Date    string // YYYY-MM-DD
    Day     int
    InMonth bool
    Today   bool
    Events  []models.Event
    More    int // events that did not fit in the box
}

type CalendarData struct {
    Title     string // "November 2026"
    Month     string // 2026-11
    PrevMonth string
    NextMonth string
    ThisMonth string
    Weekdays  []string
    Weeks     [][]CalendarDay
    Events    []models.Event // everything in the month, for the list under the grid
    IsAdmin   bool
}

// buildMonthGrid lays out a month as whole weeks (Sunday to Saturday), including the few days
// of the neighbouring months that fill the first and last row.
func buildMonthGrid(year int, month time.Month, today time.Time, events []models.Event) [][]CalendarDay {
    first := time.Date(year, month, 1, 0, 0, 0, 0, today.Location())
    start := first.AddDate(0, 0, -int(first.Weekday()))
    last := first.AddDate(0, 1, -1)
    end := last.AddDate(0, 0, 6-int(last.Weekday()))
    todayStr := dateString(today)

    var weeks [][]CalendarDay
    var week []CalendarDay
    for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
        date := dateString(d)
        cell := CalendarDay{Date: date, Day: d.Day(), InMonth: d.Month() == month, Today: date == todayStr}
        for _, e := range events {
            if e.Covers(date) {
                if len(cell.Events) < maxCellEvents {
                    cell.Events = append(cell.Events, e)
                } else {
                    cell.More++
                }
            }
        }
        week = append(week, cell)
        if len(week) == 7 {
            weeks = append(weeks, week)
            week = nil
        }
    }
    return weeks
}

// calendarScope says which exams this person may see on the calendar: admins every exam,
// everyone else only the exams of their own classes (or their children's classes).
func (a *App) calendarScope(u *SessionUser) (all bool, classIDs []int, err error) {
    switch u.Role {
    case "admin":
        return true, nil, nil
    case "teacher":
        classes, err := models.ClassesForUser(a.DB, "teacher", u.ID)
        for _, c := range classes {
            classIDs = append(classIDs, c.ID)
        }
        return false, classIDs, err
    case "student":
        s, err := models.GetStudentByUserID(a.DB, u.ID)
        if errors.Is(err, models.ErrNotFound) {
            return false, nil, nil
        }
        if err != nil {
            return false, nil, err
        }
        if s.ClassID != 0 {
            classIDs = append(classIDs, s.ClassID)
        }
        return false, classIDs, nil
    case "parent":
        p, err := models.GetParentByUserID(a.DB, u.ID)
        if errors.Is(err, models.ErrNotFound) {
            return false, nil, nil
        }
        if err != nil {
            return false, nil, err
        }
        children, err := models.ListChildren(a.DB, p.ID)
        for _, ch := range children {
            if ch.ClassID != 0 {
                classIDs = append(classIDs, ch.ClassID)
            }
        }
        return false, classIDs, err
    }
    return false, nil, nil
}

// eventsBetween returns the calendar entries this person may see.
func (a *App) eventsBetween(u *SessionUser, from, to string) ([]models.Event, error) {
    all, classIDs, err := a.calendarScope(u)
    if err != nil {
        return nil, err
    }
    return models.ListEventsBetween(a.DB, from, to, all, classIDs)
}

func (a *App) CalendarPage(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

    shown := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
    if m := r.URL.Query().Get("month"); m != "" {
        if t, err := time.ParseInLocation("2006-01", m, today.Location()); err == nil && t.Year() >= 2000 && t.Year() <= 2100 {
            shown = t
        }
    }

    weeks := buildMonthGrid(shown.Year(), shown.Month(), today, nil)
    from := weeks[0][0].Date
    to := weeks[len(weeks)-1][6].Date
    events, err := a.eventsBetween(u, from, to)
    if err != nil {
        a.serverError(w, err)
        return
    }
    weeks = buildMonthGrid(shown.Year(), shown.Month(), today, events)

    monthFirst := dateString(shown)
    monthLast := dateString(shown.AddDate(0, 1, -1))
    var inMonth []models.Event
    for _, e := range events {
        if e.EndDate >= monthFirst && e.StartDate <= monthLast {
            inMonth = append(inMonth, e)
        }
    }

    a.render(w, r, "calendar", http.StatusOK, PageData{Title: "Calendar", Data: CalendarData{
        Title:     shown.Format("January 2006"),
        Month:     shown.Format("2006-01"),
        PrevMonth: shown.AddDate(0, -1, 0).Format("2006-01"),
        NextMonth: shown.AddDate(0, 1, 0).Format("2006-01"),
        ThisMonth: today.Format("2006-01"),
        Weekdays:  []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
        Weeks:     weeks,
        Events:    inMonth,
        IsAdmin:   u.Role == "admin",
    }})
}

type EventFormData struct {
    Edit       bool
    ID         int
    Kinds      []models.EventKind
    Kind       string
    Title      string
    StartDate  string
    EndDate    string
    Details    string
    MaxTitle   int
    MaxDetails int
}

func (a *App) renderEventForm(w http.ResponseWriter, r *http.Request, status int, f EventFormData, msg string) {
    f.Kinds = models.EventKinds
    f.MaxTitle, f.MaxDetails = models.MaxEventTitle, models.MaxEventDetails
    if f.Kind == "" {
        f.Kind = "event"
    }
    title := "Add event"
    if f.Edit {
        title = "Edit event"
    }
    a.render(w, r, "event_form", status, PageData{Title: title, Error: msg, Data: f})
}

func (a *App) EventNewForm(w http.ResponseWriter, r *http.Request) {
    f := EventFormData{}
    if d := r.URL.Query().Get("date"); d != "" {
        if _, err := time.Parse("2006-01-02", d); err == nil {
            f.StartDate = d
        }
    }
    a.renderEventForm(w, r, http.StatusOK, f, "")
}

// readEvent reads and checks the event form. The message is "" when everything is fine.
func readEvent(r *http.Request) (models.Event, EventFormData, string) {
    e := models.Event{
        Kind:      r.PostFormValue("kind"),
        Title:     strings.TrimSpace(r.PostFormValue("title")),
        StartDate: r.PostFormValue("start_date"),
        EndDate:   r.PostFormValue("end_date"),
        Details:   strings.TrimSpace(r.PostFormValue("details")),
    }
    form := EventFormData{Kind: e.Kind, Title: e.Title, StartDate: e.StartDate, EndDate: e.EndDate, Details: e.Details}

    switch {
    case e.Title == "":
        return e, form, "Please give the event a title"
    case utf8.RuneCountInString(e.Title) > models.MaxEventTitle:
        return e, form, "The title is too long (max " + strconv.Itoa(models.MaxEventTitle) + " characters)"
    case utf8.RuneCountInString(e.Details) > models.MaxEventDetails:
        return e, form, "The details are too long (max " + strconv.Itoa(models.MaxEventDetails) + " characters)"
    case !models.ValidEventKind(e.Kind):
        return e, form, "Please choose what kind of event this is"
    }
    start, err := time.Parse("2006-01-02", e.StartDate)
    if err != nil {
        return e, form, "Please choose the date of the event"
    }
    if e.EndDate == "" {
        e.EndDate = e.StartDate
    }
    end, err := time.Parse("2006-01-02", e.EndDate)
    if err != nil {
        return e, form, "The end date is not a valid date"
    }
    if end.Before(start) {
        return e, form, "The end date cannot be before the start date"
    }
    if end.Sub(start).Hours()/24 >= float64(models.MaxEventDays) {
        return e, form, "An event can last at most " + strconv.Itoa(models.MaxEventDays) + " days"
    }
    return e, form, ""
}

func (a *App) EventCreate(w http.ResponseWriter, r *http.Request) {
    e, form, msg := readEvent(r)
    if msg != "" {
        a.renderEventForm(w, r, http.StatusBadRequest, form, msg)
        return
    }
    if err := models.CreateEvent(a.DB, e); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/calendar?month="+e.StartDate[:7], http.StatusSeeOther)
}

func (a *App) loadEventFromPath(w http.ResponseWriter, r *http.Request) (*models.Event, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        a.NotFound(w, r)
        return nil, false
    }
    e, err := models.GetEvent(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        a.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return e, true
}

func (a *App) EventEditForm(w http.ResponseWriter, r *http.Request) {
    e, ok := a.loadEventFromPath(w, r)
    if !ok {
        return
    }
    f := EventFormData{Edit: true, ID: e.ID, Kind: e.Kind, Title: e.Title, StartDate: e.StartDate, Details: e.Details}
    if e.MultiDay() {
        f.EndDate = e.EndDate
    }
    a.renderEventForm(w, r, http.StatusOK, f, "")
}

func (a *App) EventUpdate(w http.ResponseWriter, r *http.Request) {
    old, ok := a.loadEventFromPath(w, r)
    if !ok {
        return
    }
    e, form, msg := readEvent(r)
    if msg != "" {
        form.Edit, form.ID = true, old.ID
        a.renderEventForm(w, r, http.StatusBadRequest, form, msg)
        return
    }
    e.ID = old.ID
    if err := models.UpdateEvent(a.DB, e); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/calendar?month="+e.StartDate[:7], http.StatusSeeOther)
}

func (a *App) EventDelete(w http.ResponseWriter, r *http.Request) {
    e, ok := a.loadEventFromPath(w, r)
    if !ok {
        return
    }
    if err := models.DeleteEvent(a.DB, e.ID); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/calendar?month="+e.StartDate[:7], http.StatusSeeOther)
}

// comingUp fills the dashboard's "Coming up" list with the next few events.
func (a *App) comingUp(u *SessionUser, data *DashboardData) error {
    now := time.Now()
    from := dateString(now)
    to := dateString(now.AddDate(0, 0, 30))
    list, err := a.eventsBetween(u, from, to)
    if err != nil {
        return err
    }
    for _, e := range list {
        if e.EndDate < from {
            continue
        }
        data.Coming = append(data.Coming, e)
        if len(data.Coming) == 4 {
            break
        }
    }
    return nil
}
