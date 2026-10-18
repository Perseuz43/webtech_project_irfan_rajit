package models

import (
    "database/sql"
    "errors"
    "strings"

    "golang.org/x/crypto/bcrypt"
)

type Student struct {
    ID           int
    UserID       int
    Username     string
    Name         string
    Roll         int
    ClassID      int
    ClassName    string
    SectionID    int
    SectionName  string
    GuardianName string
    Phone        string
}

type StudentInput struct {
    Name         string
    Roll         int
    ClassID      int
    SectionID    int
    GuardianName string
    Phone        string
    Username     string
    Password     string
}

const studentSelect = `SELECT s.id, s.user_id, u.username, s.name, s.roll,
    COALESCE(s.class_id, 0), COALESCE(c.name, ''),
    COALESCE(s.section_id, 0), COALESCE(sec.name, ''),
    s.guardian_name, s.phone
    FROM students s
    JOIN users u ON u.id = s.user_id
    LEFT JOIN classes c ON c.id = s.class_id
    LEFT JOIN sections sec ON sec.id = s.section_id`

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
    Scan(dest ...any) error
}

func scanStudent(row rowScanner) (*Student, error) {
    s := &Student{}
    err := row.Scan(&s.ID, &s.UserID, &s.Username, &s.Name, &s.Roll,
        &s.ClassID, &s.ClassName, &s.SectionID, &s.SectionName,
        &s.GuardianName, &s.Phone)
    if err != nil {
        return nil, err
    }
    return s, nil
}

func notFoundIf(err error) error {
    if errors.Is(err, sql.ErrNoRows) {
        return ErrNotFound
    }
    return err
}

// nullInt stores 0 as NULL.
func nullInt(n int) any {
    if n == 0 {
        return nil
    }
    return n
}

func hashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

// IsDuplicateOn reports whether err is a UNIQUE violation on the given column, e.g. "users.username".
func IsDuplicateOn(err error, column string) bool {
    return IsDuplicate(err) && strings.Contains(err.Error(), column)
}

func ListStudents(db *sql.DB) ([]Student, error) {
    rows, err := db.Query(studentSelect +
        " ORDER BY COALESCE(s.class_id, 0), COALESCE(sec.name, ''), s.roll")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Student
    for rows.Next() {
        s, err := scanStudent(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *s)
    }
    return list, rows.Err()
}

func GetStudent(db *sql.DB, id int) (*Student, error) {
    s, err := scanStudent(db.QueryRow(studentSelect+" WHERE s.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return s, nil
}

func GetStudentByUserID(db *sql.DB, userID int) (*Student, error) {
    s, err := scanStudent(db.QueryRow(studentSelect+" WHERE s.user_id = ?", userID))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return s, nil
}

// CreateStudent creates the login account and the student profile in one transaction.
func CreateStudent(db *sql.DB, in StudentInput) error {
    hash, err := hashPassword(in.Password)
    if err != nil {
        return err
    }
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    res, err := tx.Exec(
        "INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'student')",
        in.Username, hash)
    if err != nil {
        return err
    }
    userID, err := res.LastInsertId()
    if err != nil {
        return err
    }

    _, err = tx.Exec(`INSERT INTO students (user_id, name, roll, class_id, section_id, guardian_name, phone)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
        userID, in.Name, in.Roll, nullInt(in.ClassID), nullInt(in.SectionID), in.GuardianName, in.Phone)
    if err != nil {
        return err
    }
    return tx.Commit()
}

func UpdateStudent(db *sql.DB, id int, in StudentInput) error {
    _, err := db.Exec(`UPDATE students
        SET name = ?, roll = ?, class_id = ?, section_id = ?, guardian_name = ?, phone = ?
        WHERE id = ?`,
        in.Name, in.Roll, nullInt(in.ClassID), nullInt(in.SectionID), in.GuardianName, in.Phone, id)
    return err
}

// DeleteStudent removes the login account; the student profile is removed with it.
func DeleteStudent(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM users WHERE id = (SELECT user_id FROM students WHERE id = ?)", id)
    return err
}

func ListAllSections(db *sql.DB) ([]Section, error) {
    rows, err := db.Query("SELECT id, class_id, name FROM sections ORDER BY class_id, name")
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

func SectionBelongsToClass(db *sql.DB, sectionID, classID int) (bool, error) {
    var n int
    err := db.QueryRow("SELECT COUNT(*) FROM sections WHERE id = ? AND class_id = ?",
        sectionID, classID).Scan(&n)
    return n > 0, err
}
