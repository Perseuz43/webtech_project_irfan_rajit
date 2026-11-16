package models

import (
    "database/sql"
    "errors"
)

// AcademicYear is a school year. Exams, fees and attendance are not stored "inside" a year:
// each is dated, and it belongs to the year whose dates contain that date.
type AcademicYear struct {
    ID        int
    Name      string
    StartDate string // YYYY-MM-DD
    EndDate   string
    IsCurrent bool
}

// Contains says whether a date (YYYY-MM-DD) falls inside the year.
func (y AcademicYear) Contains(date string) bool {
    return y.StartDate <= date && date <= y.EndDate
}

const yearSelect = "SELECT id, name, start_date, end_date, is_current FROM academic_years"

func scanYear(row interface{ Scan(...any) error }) (*AcademicYear, error) {
    var y AcademicYear
    var cur int
    if err := row.Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &cur); err != nil {
        return nil, err
    }
    y.IsCurrent = cur == 1
    return &y, nil
}

// ListYears returns every year, the newest first.
func ListYears(db *sql.DB) ([]AcademicYear, error) {
    rows, err := db.Query(yearSelect + " ORDER BY start_date DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []AcademicYear
    for rows.Next() {
        y, err := scanYear(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *y)
    }
    return list, rows.Err()
}

func GetYear(db *sql.DB, id int) (*AcademicYear, error) {
    y, err := scanYear(db.QueryRow(yearSelect+" WHERE id = ?", id))
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return y, err
}

// YearOverlaps tells whether any existing year shares a day with start..end.
func YearOverlaps(db *sql.DB, start, end string) (*AcademicYear, error) {
    y, err := scanYear(db.QueryRow(yearSelect+" WHERE start_date <= ? AND end_date >= ? LIMIT 1", end, start))
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil
    }
    return y, err
}

// CreateYear adds a year. The very first year becomes the current one automatically.
func CreateYear(db *sql.DB, name, start, end string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var count int
    if err := tx.QueryRow("SELECT COUNT(*) FROM academic_years WHERE is_current = 1").Scan(&count); err != nil {
        return err
    }
    current := 0
    if count == 0 {
        current = 1
    }
    if _, err := tx.Exec("INSERT INTO academic_years (name, start_date, end_date, is_current) VALUES (?, ?, ?, ?)",
        name, start, end, current); err != nil {
        return err
    }
    return tx.Commit()
}

// SetCurrentYear makes one year the current one (and every other year not current).
func SetCurrentYear(db *sql.DB, id int) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.Exec("UPDATE academic_years SET is_current = 0"); err != nil {
        return err
    }
    if _, err := tx.Exec("UPDATE academic_years SET is_current = 1 WHERE id = ?", id); err != nil {
        return err
    }
    return tx.Commit()
}

// DeleteYear removes a year. Nothing else is deleted: exams, fees and attendance keep their dates.
func DeleteYear(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM academic_years WHERE id = ?", id)
    return err
}

// DefaultYear picks the year to work in when nobody chose one: the year marked current, or
// else the one that contains today. It returns nil when no year fits.
func DefaultYear(years []AcademicYear, today string) *AcademicYear {
    for i := range years {
        if years[i].IsCurrent {
            return &years[i]
        }
    }
    for i := range years {
        if years[i].Contains(today) {
            return &years[i]
        }
    }
    return nil
}
