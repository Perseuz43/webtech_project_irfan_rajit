package models

import (
    "database/sql"
    "errors"
    "sort"
    "strings"
)

const (
    MaxEventTitle   = 100
    MaxEventDetails = 500
    MaxEventDays    = 60 // the longest event, in days
)

type EventKind struct {
    Value string
    Label string
}

var EventKinds = []EventKind{
    {"event", "Event"},
    {"holiday", "Holiday"},
    {"meeting", "Meeting"},
    {"exam", "Exam"},
}

func ValidEventKind(k string) bool {
    for _, e := range EventKinds {
        if e.Value == k {
            return true
        }
    }
    return false
}

// Event is something on the school calendar. Exams that are set up under Exams appear on the
// calendar by themselves (Auto is true); those cannot be changed here.
type Event struct {
    ID        int
    Title     string
    Kind      string
    StartDate string // YYYY-MM-DD
    EndDate   string // the same as StartDate for a one-day event
    Details   string
    Auto      bool
}

func (e Event) KindLabel() string {
    for _, k := range EventKinds {
        if k.Value == e.Kind {
            return k.Label
        }
    }
    return "Event"
}

// Covers says whether the event is on the given day.
func (e Event) Covers(date string) bool { return e.StartDate <= date && date <= e.EndDate }

func (e Event) MultiDay() bool { return e.EndDate != e.StartDate }

func CreateEvent(db *sql.DB, e Event) error {
    _, err := db.Exec("INSERT INTO events (title, kind, start_date, end_date, details) VALUES (?, ?, ?, ?, ?)",
        e.Title, e.Kind, e.StartDate, e.EndDate, e.Details)
    return err
}

func GetEvent(db *sql.DB, id int) (*Event, error) {
    var e Event
    err := db.QueryRow("SELECT id, title, kind, start_date, end_date, details FROM events WHERE id = ?", id).
        Scan(&e.ID, &e.Title, &e.Kind, &e.StartDate, &e.EndDate, &e.Details)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return &e, err
}

func UpdateEvent(db *sql.DB, e Event) error {
    _, err := db.Exec("UPDATE events SET title = ?, kind = ?, start_date = ?, end_date = ?, details = ? WHERE id = ?",
        e.Title, e.Kind, e.StartDate, e.EndDate, e.Details, e.ID)
    return err
}

func DeleteEvent(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM events WHERE id = ?", id)
    return err
}

// ListEventsBetween returns the events that touch the days from..to (inclusive).
// Exam dates from the exams table are added too: for every class when allClasses is true,
// otherwise only for the given classes.
func ListEventsBetween(db *sql.DB, from, to string, allClasses bool, classIDs []int) ([]Event, error) {
    rows, err := db.Query(`SELECT id, title, kind, start_date, end_date, details FROM events
        WHERE end_date >= ? AND start_date <= ?`, from, to)
    if err != nil {
        return nil, err
    }
    var list []Event
    for rows.Next() {
        var e Event
        if err := rows.Scan(&e.ID, &e.Title, &e.Kind, &e.StartDate, &e.EndDate, &e.Details); err != nil {
            rows.Close()
            return nil, err
        }
        list = append(list, e)
    }
    rows.Close()
    if err := rows.Err(); err != nil {
        return nil, err
    }

    query := `SELECT e.name, c.name, e.exam_date FROM exams e JOIN classes c ON c.id = e.class_id
        WHERE e.exam_date >= ? AND e.exam_date <= ?`
    args := []any{from, to}
    if !allClasses {
        if len(classIDs) == 0 {
            sortEvents(list)
            return list, nil
        }
        query += " AND e.class_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(classIDs)), ",") + ")"
        for _, id := range classIDs {
            args = append(args, id)
        }
    }
    exams, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer exams.Close()
    for exams.Next() {
        var name, class, date string
        if err := exams.Scan(&name, &class, &date); err != nil {
            return nil, err
        }
        list = append(list, Event{Title: name + " (" + class + ")", Kind: "exam", StartDate: date, EndDate: date, Auto: true})
    }
    if err := exams.Err(); err != nil {
        return nil, err
    }
    sortEvents(list)
    return list, nil
}

func sortEvents(list []Event) {
    sort.SliceStable(list, func(i, j int) bool {
        if list[i].StartDate != list[j].StartDate {
            return list[i].StartDate < list[j].StartDate
        }
        return list[i].Title < list[j].Title
    })
}
