package models

import (
    "database/sql"
    "errors"
    "sort"
    "strings"
    "time"
)

// BirthdayEntry is one student's next birthday.
type BirthdayEntry struct {
    StudentID    int
    Roll         int
    Name         string
    ClassName    string
    SectionName  string
    PhotoVersion string
    DOB          string    // YYYY-MM-DD
    Next         time.Time // the next birthday on or after today
    DaysUntil    int
    Turning      int // the age they will be
}

func (b BirthdayEntry) Today() bool { return b.DaysUntil == 0 }

func (b BirthdayEntry) DateLabel() string { return b.Next.Format("Mon 2 Jan") }

func (b BirthdayEntry) Initial() string {
    if runes := []rune(strings.TrimSpace(b.Name)); len(runes) > 0 {
        return strings.ToUpper(string(runes[0]))
    }
    return "?"
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

// birthdayInYear is the dob's birthday in the given year. A 29 February birthday is
// celebrated on 28 February in years that have no 29 February.
func birthdayInYear(dob time.Time, year int, loc *time.Location) time.Time {
    month, day := dob.Month(), dob.Day()
    if month == time.February && day == 29 && !isLeap(year) {
        day = 28
    }
    return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// nextBirthday returns the first birthday that is today or later, and the age turned.
func nextBirthday(dob, today time.Time) (time.Time, int) {
    next := birthdayInYear(dob, today.Year(), today.Location())
    if next.Before(today) {
        next = birthdayInYear(dob, today.Year()+1, today.Location())
    }
    return next, next.Year() - dob.Year()
}

// daysBetween counts whole calendar days from a to b (both at midnight).
func daysBetween(a, b time.Time) int {
    ay, am, ad := a.Date()
    by, bm, bd := b.Date()
    da := time.Date(ay, am, ad, 12, 0, 0, 0, time.UTC)
    db := time.Date(by, bm, bd, 12, 0, 0, 0, time.UTC)
    return int(db.Sub(da).Hours() / 24)
}

// ValidDOB checks a date of birth typed as YYYY-MM-DD: a real date, not in the future,
// and not absurdly old.
func ValidDOB(s string, today time.Time) bool {
    t, err := time.ParseInLocation("2006-01-02", s, today.Location())
    if err != nil {
        return false
    }
    return !t.After(today) && t.Year() >= 1920
}

func SetStudentDOB(db *sql.DB, studentID int, dob string) error {
    if dob == "" {
        _, err := db.Exec("DELETE FROM student_details WHERE student_id = ?", studentID)
        return err
    }
    _, err := db.Exec(`INSERT INTO student_details (student_id, dob) VALUES (?, ?)
        ON CONFLICT(student_id) DO UPDATE SET dob = excluded.dob`, studentID, dob)
    return err
}

// GetStudentDOB returns "" when no date of birth is stored.
func GetStudentDOB(db *sql.DB, studentID int) (string, error) {
    var dob string
    err := db.QueryRow("SELECT dob FROM student_details WHERE student_id = ?", studentID).Scan(&dob)
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil
    }
    return dob, err
}

// StudentBirthday returns one student's next birthday, or nil when no date of birth is stored.
func StudentBirthday(db *sql.DB, studentID int, today time.Time) (*BirthdayEntry, error) {
    var e BirthdayEntry
    err := db.QueryRow(`SELECT s.id, s.roll, s.name, COALESCE(c.name, ''), COALESCE(sec.name, ''), d.dob
        FROM student_details d
        JOIN students s ON s.id = d.student_id
        LEFT JOIN classes c ON c.id = s.class_id
        LEFT JOIN sections sec ON sec.id = s.section_id
        WHERE d.student_id = ? AND d.dob <> ''`, studentID).
        Scan(&e.StudentID, &e.Roll, &e.Name, &e.ClassName, &e.SectionName, &e.DOB)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    dob, err := time.ParseInLocation("2006-01-02", e.DOB, today.Location())
    if err != nil {
        return nil, nil
    }
    e.Next, e.Turning = nextBirthday(dob, today)
    e.DaysUntil = daysBetween(today, e.Next)
    return &e, nil
}

// CountStudentsWithoutDOB tells the admin how many students still need a date of birth.
func CountStudentsWithoutDOB(db *sql.DB) (int, error) {
    var n int
    err := db.QueryRow(`SELECT COUNT(*) FROM students s
        WHERE NOT EXISTS (SELECT 1 FROM student_details d WHERE d.student_id = s.id AND d.dob <> '')`).Scan(&n)
    return n, err
}

// ListBirthdays returns birthdays happening from today within the next withinDays days,
// soonest first. Admins see every student; a teacher only sees students of the classes
// where they teach.
func ListBirthdays(db *sql.DB, role string, userID int, today time.Time, withinDays int) ([]BirthdayEntry, error) {
    query := `SELECT s.id, s.roll, s.name, COALESCE(c.name, ''), COALESCE(sec.name, ''),
        COALESCE(ph.version, ''), d.dob
        FROM student_details d
        JOIN students s ON s.id = d.student_id
        LEFT JOIN classes c ON c.id = s.class_id
        LEFT JOIN sections sec ON sec.id = s.section_id
        LEFT JOIN student_photos ph ON ph.student_id = s.id
        WHERE d.dob <> ''`
    var args []any
    if role != "admin" {
        query += ` AND s.class_id IN (
            SELECT b.class_id FROM teacher_subjects ts
            JOIN subjects b ON b.id = ts.subject_id
            JOIN teachers t ON t.id = ts.teacher_id
            WHERE t.user_id = ?)`
        args = append(args, userID)
    }

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []BirthdayEntry
    for rows.Next() {
        var e BirthdayEntry
        if err := rows.Scan(&e.StudentID, &e.Roll, &e.Name, &e.ClassName, &e.SectionName, &e.PhotoVersion, &e.DOB); err != nil {
            return nil, err
        }
        dob, err := time.ParseInLocation("2006-01-02", e.DOB, today.Location())
        if err != nil {
            continue // ignore a damaged value instead of failing the whole page
        }
        e.Next, e.Turning = nextBirthday(dob, today)
        e.DaysUntil = daysBetween(today, e.Next)
        if e.DaysUntil <= withinDays {
            list = append(list, e)
        }
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    sort.SliceStable(list, func(i, j int) bool {
        if list[i].DaysUntil != list[j].DaysUntil {
            return list[i].DaysUntil < list[j].DaysUntil
        }
        if list[i].ClassName != list[j].ClassName {
            return list[i].ClassName < list[j].ClassName
        }
        return list[i].Roll < list[j].Roll
    })
    return list, nil
}
