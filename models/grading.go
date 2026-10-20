package models

import (
    "database/sql"
    "errors"
    "fmt"
    "math"
    "sort"
    "strconv"
    "strings"
)

type GradeBand struct {
    MinPercent float64
    Grade      string
    Point      float64
}

type GradingScale struct {
    ID          int
    Name        string
    PassPercent float64
    UseGPA      bool
    IsPreset    bool
    Bands       []GradeBand // highest first
}

type ScaleInput struct {
    Name        string
    PassPercent float64
    UseGPA      bool
    Bands       []GradeBand
}

// Grade returns the grade and grade point for a percentage ("-" and 0 if the scale has no bands).
func (s *GradingScale) Grade(percent float64) (string, float64) {
    for _, b := range s.Bands {
        if percent+1e-9 >= b.MinPercent {
            return b.Grade, b.Point
        }
    }
    return "-", 0
}

func (s *GradingScale) Passed(percent float64) bool {
    return percent+1e-9 >= s.PassPercent
}

// BandsText turns the bands back into the one-line-per-band text used on the edit form.
func (s *GradingScale) BandsText() string {
    var lines []string
    for _, b := range s.Bands {
        line := fmt.Sprintf("%g %s", b.MinPercent, b.Grade)
        if s.UseGPA {
            line += fmt.Sprintf(" %.2f", b.Point)
        }
        lines = append(lines, line)
    }
    return strings.Join(lines, "\n")
}

// ParseBands reads one band per line: "minimum-percent grade [grade-point]".
func ParseBands(text string, useGPA bool) ([]GradeBand, error) {
    var bands []GradeBand
    seen := map[float64]bool{}

    for i, line := range strings.Split(text, "\n") {
        fields := strings.Fields(line)
        if len(fields) == 0 {
            continue
        }
        n := i + 1
        want := 2
        if useGPA {
            want = 3
        }
        if len(fields) != want {
            if useGPA {
                return nil, fmt.Errorf("Line %d: write it as: minimum-percent grade grade-point (for example: 80 A+ 5.00)", n)
            }
            return nil, fmt.Errorf("Line %d: write it as: minimum-percent grade (for example: 80 A)", n)
        }

        min, err := strconv.ParseFloat(fields[0], 64)
        if err != nil || math.IsNaN(min) || min < 0 || min > 100 {
            return nil, fmt.Errorf("Line %d: the minimum percent must be a number from 0 to 100", n)
        }
        grade := fields[1]
        if len([]rune(grade)) > 10 {
            return nil, fmt.Errorf("Line %d: the grade is too long (max 10 characters)", n)
        }
        point := 0.0
        if useGPA {
            point, err = strconv.ParseFloat(fields[2], 64)
            if err != nil || math.IsNaN(point) || point < 0 || point > 10 {
                return nil, fmt.Errorf("Line %d: the grade point must be a number from 0 to 10", n)
            }
        }
        if seen[min] {
            return nil, fmt.Errorf("Line %d: another line already starts at %g", n, min)
        }
        seen[min] = true
        bands = append(bands, GradeBand{MinPercent: min, Grade: grade, Point: point})
    }

    if len(bands) > 20 {
        return nil, errors.New("Use at most 20 grade lines")
    }
    if len(bands) > 0 && !seen[0] {
        return nil, errors.New("One line must start at 0, so that every mark gets a grade")
    }
    sort.Slice(bands, func(i, j int) bool { return bands[i].MinPercent > bands[j].MinPercent })
    return bands, nil
}

func loadBands(db *sql.DB, s *GradingScale) error {
    rows, err := db.Query(
        "SELECT min_percent, grade, point FROM grade_bands WHERE scale_id = ? ORDER BY min_percent DESC", s.ID)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var b GradeBand
        if err := rows.Scan(&b.MinPercent, &b.Grade, &b.Point); err != nil {
            return err
        }
        s.Bands = append(s.Bands, b)
    }
    return rows.Err()
}

func ListScales(db *sql.DB) ([]*GradingScale, error) {
    rows, err := db.Query(
        "SELECT id, name, pass_percent, use_gpa, is_preset FROM grading_scales ORDER BY is_preset DESC, id")
    if err != nil {
        return nil, err
    }
    var list []*GradingScale
    for rows.Next() {
        s := &GradingScale{}
        var gpa, preset int
        if err := rows.Scan(&s.ID, &s.Name, &s.PassPercent, &gpa, &preset); err != nil {
            rows.Close()
            return nil, err
        }
        s.UseGPA = gpa == 1
        s.IsPreset = preset == 1
        list = append(list, s)
    }
    if err := rows.Err(); err != nil {
        rows.Close()
        return nil, err
    }
    rows.Close()

    for _, s := range list {
        if err := loadBands(db, s); err != nil {
            return nil, err
        }
    }
    return list, nil
}

func GetScale(db *sql.DB, id int) (*GradingScale, error) {
    s := &GradingScale{}
    var gpa, preset int
    err := db.QueryRow(
        "SELECT id, name, pass_percent, use_gpa, is_preset FROM grading_scales WHERE id = ?", id).
        Scan(&s.ID, &s.Name, &s.PassPercent, &gpa, &preset)
    if err != nil {
        return nil, notFoundIf(err)
    }
    s.UseGPA = gpa == 1
    s.IsPreset = preset == 1
    if err := loadBands(db, s); err != nil {
        return nil, err
    }
    return s, nil
}

// ActiveScale returns the grading system the school currently uses.
func ActiveScale(db *sql.DB) (*GradingScale, error) {
    var v string
    err := db.QueryRow("SELECT value FROM settings WHERE key = 'active_grading_scale'").Scan(&v)
    if err == nil {
        if id, convErr := strconv.Atoi(v); convErr == nil {
            if s, getErr := GetScale(db, id); getErr == nil {
                return s, nil
            }
        }
    } else if !errors.Is(err, sql.ErrNoRows) {
        return nil, err
    }

    // nothing chosen (or the chosen one is gone): use the first system
    var id int
    if err := db.QueryRow("SELECT id FROM grading_scales ORDER BY id LIMIT 1").Scan(&id); err != nil {
        return nil, notFoundIf(err)
    }
    return GetScale(db, id)
}

func SetActiveScale(db *sql.DB, id int) error {
    if _, err := GetScale(db, id); err != nil {
        return err
    }
    _, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('active_grading_scale', ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value`, strconv.Itoa(id))
    return err
}

// SaveScale creates a custom scale (id 0) or updates one. Built-in scales are never changed.
func SaveScale(db *sql.DB, id int, in ScaleInput) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    gpa := 0
    if in.UseGPA {
        gpa = 1
    }
    if id == 0 {
        res, err := tx.Exec(
            "INSERT INTO grading_scales (name, pass_percent, use_gpa, is_preset) VALUES (?, ?, ?, 0)",
            in.Name, in.PassPercent, gpa)
        if err != nil {
            return err
        }
        newID, err := res.LastInsertId()
        if err != nil {
            return err
        }
        id = int(newID)
    } else {
        _, err := tx.Exec(
            "UPDATE grading_scales SET name = ?, pass_percent = ?, use_gpa = ? WHERE id = ? AND is_preset = 0",
            in.Name, in.PassPercent, gpa, id)
        if err != nil {
            return err
        }
        if _, err := tx.Exec("DELETE FROM grade_bands WHERE scale_id = ?", id); err != nil {
            return err
        }
    }

    for _, b := range in.Bands {
        _, err := tx.Exec("INSERT INTO grade_bands (scale_id, min_percent, grade, point) VALUES (?, ?, ?, ?)",
            id, b.MinPercent, b.Grade, b.Point)
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}

func DeleteScale(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM grading_scales WHERE id = ? AND is_preset = 0", id)
    return err
}

// SeedGrading adds the built-in grading systems the first time the app runs.
func SeedGrading(db *sql.DB) error {
    var n int
    if err := db.QueryRow("SELECT COUNT(*) FROM grading_scales").Scan(&n); err != nil {
        return err
    }
    if n > 0 {
        return nil
    }

    presets := []struct {
        name  string
        pass  float64
        gpa   bool
        bands []GradeBand
    }{
        {"Bangladesh GPA (5.00 scale)", 33, true, []GradeBand{
            {80, "A+", 5}, {70, "A", 4}, {60, "A-", 3.5}, {50, "B", 3}, {40, "C", 2}, {33, "D", 1}, {0, "F", 0}}},
        {"4.0 GPA scale (A-F)", 60, true, []GradeBand{
            {90, "A", 4}, {80, "B", 3}, {70, "C", 2}, {60, "D", 1}, {0, "F", 0}}},
        {"Letter grades (A-F, no GPA)", 60, false, []GradeBand{
            {90, "A", 0}, {80, "B", 0}, {70, "C", 0}, {60, "D", 0}, {0, "F", 0}}},
        {"Percentage only (no grades)", 40, false, nil},
    }

    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    firstID := 0
    for i, p := range presets {
        gpa := 0
        if p.gpa {
            gpa = 1
        }
        res, err := tx.Exec(
            "INSERT INTO grading_scales (name, pass_percent, use_gpa, is_preset) VALUES (?, ?, ?, 1)",
            p.name, p.pass, gpa)
        if err != nil {
            return err
        }
        id, err := res.LastInsertId()
        if err != nil {
            return err
        }
        if i == 0 {
            firstID = int(id)
        }
        for _, b := range p.bands {
            _, err := tx.Exec("INSERT INTO grade_bands (scale_id, min_percent, grade, point) VALUES (?, ?, ?, ?)",
                id, b.MinPercent, b.Grade, b.Point)
            if err != nil {
                return err
            }
        }
    }
    _, err = tx.Exec("INSERT INTO settings (key, value) VALUES ('active_grading_scale', ?)", strconv.Itoa(firstID))
    if err != nil {
        return err
    }
    return tx.Commit()
}
