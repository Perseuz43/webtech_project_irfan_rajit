package models

import (
    "database/sql"
    "errors"
    "strings"
)

type Class struct {
    ID           int
    Name         string
    SectionCount int
    SubjectCount int
}

type Section struct {
    ID      int
    ClassID int
    Name    string
}

type Subject struct {
    ID      int
    ClassID int
    Name    string
}

// IsDuplicate reports whether err is a UNIQUE constraint violation.
func IsDuplicate(err error) bool {
    return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ---------- classes ----------

func ListClasses(db *sql.DB) ([]Class, error) {
    rows, err := db.Query(`SELECT c.id, c.name,
        (SELECT COUNT(*) FROM sections s WHERE s.class_id = c.id),
        (SELECT COUNT(*) FROM subjects b WHERE b.class_id = c.id)
        FROM classes c ORDER BY c.id`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Class
    for rows.Next() {
        var c Class
        if err := rows.Scan(&c.ID, &c.Name, &c.SectionCount, &c.SubjectCount); err != nil {
            return nil, err
        }
        list = append(list, c)
    }
    return list, rows.Err()
}

func GetClass(db *sql.DB, id int) (*Class, error) {
    c := &Class{}
    err := db.QueryRow("SELECT id, name FROM classes WHERE id = ?", id).Scan(&c.ID, &c.Name)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    return c, nil
}

func CreateClass(db *sql.DB, name string) error {
    _, err := db.Exec("INSERT INTO classes (name) VALUES (?)", name)
    return err
}

func RenameClass(db *sql.DB, id int, name string) error {
    _, err := db.Exec("UPDATE classes SET name = ? WHERE id = ?", name, id)
    return err
}

func DeleteClass(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM classes WHERE id = ?", id)
    return err
}

// ---------- sections ----------

func ListSections(db *sql.DB, classID int) ([]Section, error) {
    rows, err := db.Query("SELECT id, class_id, name FROM sections WHERE class_id = ? ORDER BY name", classID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Section
    for rows.Next() {
        var s Section
        if err := rows.Scan(&s.ID, &s.ClassID, &s.Name); err != nil {
            return nil, err
        }
        list = append(list, s)
    }
    return list, rows.Err()
}

func CreateSection(db *sql.DB, classID int, name string) error {
    _, err := db.Exec("INSERT INTO sections (class_id, name) VALUES (?, ?)", classID, name)
    return err
}

func DeleteSection(db *sql.DB, classID, id int) error {
    _, err := db.Exec("DELETE FROM sections WHERE id = ? AND class_id = ?", id, classID)
    return err
}

// ---------- subjects ----------

func ListSubjects(db *sql.DB, classID int) ([]Subject, error) {
    rows, err := db.Query("SELECT id, class_id, name FROM subjects WHERE class_id = ? ORDER BY name", classID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Subject
    for rows.Next() {
        var s Subject
        if err := rows.Scan(&s.ID, &s.ClassID, &s.Name); err != nil {
            return nil, err
        }
        list = append(list, s)
    }
    return list, rows.Err()
}

func CreateSubject(db *sql.DB, classID int, name string) error {
    _, err := db.Exec("INSERT INTO subjects (class_id, name) VALUES (?, ?)", classID, name)
    return err
}

func DeleteSubject(db *sql.DB, classID, id int) error {
    _, err := db.Exec("DELETE FROM subjects WHERE id = ? AND class_id = ?", id, classID)
    return err
}
