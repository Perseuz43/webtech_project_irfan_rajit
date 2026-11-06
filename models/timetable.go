package models

import (
    "database/sql"
    "fmt"
    "regexp"
)

const (
    MaxPeriods     = 12 // the most periods a day can have
    DefaultPeriods = 8  // rows shown in the editor before any are used
)

// DayNames is indexed by day number: 0 = Sunday ... 6 = Saturday (the same as Go's time.Weekday).
var DayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

type Slot struct {
    ID          int
    ClassID     int
    ClassName   string
    Day         int
    Period      int
    SubjectID   int
    SubjectName string
    TeacherID   int // 0 = no teacher chosen
    TeacherName string
}

type SlotInput struct {
    Day       int
    Period    int
    SubjectID int
    TeacherID int
}

type PeriodTime struct {
    Period int
    Start  string // "08:00"
    End    string
}

// ConflictError is returned when a teacher would be in two classes at the same time.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

var clockRe = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// ValidClock accepts a 24-hour time such as "08:30".
func ValidClock(s string) bool { return clockRe.MatchString(s) }

const slotSelect = `SELECT ts.id, ts.class_id, c.name, ts.day, ts.period, ts.subject_id, b.name,
    COALESCE(ts.teacher_id, 0), COALESCE(t.name, '')
    FROM timetable_slots ts
    JOIN classes c ON c.id = ts.class_id
    JOIN subjects b ON b.id = ts.subject_id
    LEFT JOIN teachers t ON t.id = ts.teacher_id`

func querySlots(db *sql.DB, query string, args ...any) ([]Slot, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Slot
    for rows.Next() {
        var s Slot
        if err := rows.Scan(&s.ID, &s.ClassID, &s.ClassName, &s.Day, &s.Period, &s.SubjectID, &s.SubjectName,
            &s.TeacherID, &s.TeacherName); err != nil {
            return nil, err
        }
        list = append(list, s)
    }
    return list, rows.Err()
}

// ListClassSlots returns every lesson of one class, in week order.
func ListClassSlots(db *sql.DB, classID int) ([]Slot, error) {
    return querySlots(db, slotSelect+" WHERE ts.class_id = ? ORDER BY ts.day, ts.period", classID)
}

// ListTeacherSlots returns every lesson a teacher gives, in week order.
func ListTeacherSlots(db *sql.DB, teacherID int) ([]Slot, error) {
    return querySlots(db, slotSelect+" WHERE ts.teacher_id = ? ORDER BY ts.day, ts.period", teacherID)
}

// ListClassSlotsOnDay returns one class's lessons for a single weekday (0 = Sunday).
func ListClassSlotsOnDay(db *sql.DB, classID, day int) ([]Slot, error) {
    return querySlots(db, slotSelect+" WHERE ts.class_id = ? AND ts.day = ? ORDER BY ts.period", classID, day)
}

// ListTeacherSlotsOnDay returns one teacher's lessons for a single weekday (0 = Sunday).
func ListTeacherSlotsOnDay(db *sql.DB, teacherID, day int) ([]Slot, error) {
    return querySlots(db, slotSelect+" WHERE ts.teacher_id = ? AND ts.day = ? ORDER BY ts.period", teacherID, day)
}

// SaveClassTimetable replaces a class's whole timetable. If a teacher is already teaching
// another class in the same period, nothing is saved and a *ConflictError is returned.
func SaveClassTimetable(db *sql.DB, classID int, slots []SlotInput) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.Exec("DELETE FROM timetable_slots WHERE class_id = ?", classID); err != nil {
        return err
    }
    for _, s := range slots {
        if s.TeacherID > 0 {
            var otherClass, teacher string
            err := tx.QueryRow(`SELECT c.name, t.name FROM timetable_slots ts
                JOIN classes c ON c.id = ts.class_id
                JOIN teachers t ON t.id = ts.teacher_id
                WHERE ts.teacher_id = ? AND ts.day = ? AND ts.period = ? AND ts.class_id <> ?`,
                s.TeacherID, s.Day, s.Period, classID).Scan(&otherClass, &teacher)
            if err == nil {
                return &ConflictError{Message: fmt.Sprintf("%s is already teaching %s on %s, period %d",
                    teacher, otherClass, DayNames[s.Day], s.Period)}
            }
            if err != sql.ErrNoRows {
                return err
            }
        }
        if _, err := tx.Exec(`INSERT INTO timetable_slots (class_id, day, period, subject_id, teacher_id)
            VALUES (?, ?, ?, ?, ?)`, classID, s.Day, s.Period, s.SubjectID, nullInt(s.TeacherID)); err != nil {
            return err
        }
    }
    return tx.Commit()
}

// ListPeriodTimes returns the clock times of the periods, keyed by period number.
func ListPeriodTimes(db *sql.DB) (map[int]PeriodTime, error) {
    rows, err := db.Query("SELECT period, start_time, end_time FROM timetable_periods")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    m := make(map[int]PeriodTime)
    for rows.Next() {
        var p PeriodTime
        if err := rows.Scan(&p.Period, &p.Start, &p.End); err != nil {
            return nil, err
        }
        m[p.Period] = p
    }
    return m, rows.Err()
}

// SavePeriodTimes replaces the period times. A period with no times is simply not stored.
func SavePeriodTimes(db *sql.DB, times []PeriodTime) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.Exec("DELETE FROM timetable_periods"); err != nil {
        return err
    }
    for _, p := range times {
        if _, err := tx.Exec("INSERT INTO timetable_periods (period, start_time, end_time) VALUES (?, ?, ?)",
            p.Period, p.Start, p.End); err != nil {
            return err
        }
    }
    return tx.Commit()
}
