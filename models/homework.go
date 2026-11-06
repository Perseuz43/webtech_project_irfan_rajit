package models

import (
    "database/sql"
    "errors"
    "strings"
)

const (
    MaxHomeworkTitle   = 100
    MaxHomeworkDetails = 1000
)

type Homework struct {
    ID          int
    ClassID     int
    ClassName   string
    SubjectID   int
    SubjectName string
    Title       string
    Details     string
    DueDate     string // YYYY-MM-DD
    PostedBy    int    // user ID, 0 if that account was deleted
    PosterName  string
    Editable    bool // set by the page: may the viewer change or delete this?
}

type HomeworkInput struct {
    ClassID   int
    SubjectID int
    Title     string
    Details   string
    DueDate   string
    PostedBy  int
}

// HomeworkSubject is a class + subject pair that someone may post homework for.
type HomeworkSubject struct {
    SubjectID   int
    SubjectName string
    ClassID     int
    ClassName   string
}

// HomeworkSubjects lists where a user may post homework: an admin anywhere, a teacher only
// for the subjects the admin has assigned to them.
func HomeworkSubjects(db *sql.DB, role string, userID int) ([]HomeworkSubject, error) {
    query := `SELECT b.id, b.name, c.id, c.name FROM subjects b JOIN classes c ON c.id = b.class_id`
    var args []any
    if role != "admin" {
        query += ` WHERE b.id IN (SELECT ts.subject_id FROM teacher_subjects ts
            JOIN teachers t ON t.id = ts.teacher_id WHERE t.user_id = ?)`
        args = append(args, userID)
    }
    query += " ORDER BY c.name, b.name"

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []HomeworkSubject
    for rows.Next() {
        var s HomeworkSubject
        if err := rows.Scan(&s.SubjectID, &s.SubjectName, &s.ClassID, &s.ClassName); err != nil {
            return nil, err
        }
        list = append(list, s)
    }
    return list, rows.Err()
}

const homeworkSelect = `SELECT h.id, h.class_id, c.name, h.subject_id, b.name, h.title, h.details, h.due_date,
    COALESCE(h.posted_by, 0), COALESCE(t.name, CASE WHEN h.posted_by IS NULL THEN '' ELSE 'Admin' END)
    FROM homework h
    JOIN classes c ON c.id = h.class_id
    JOIN subjects b ON b.id = h.subject_id
    LEFT JOIN teachers t ON t.user_id = h.posted_by`

func scanHomework(row interface{ Scan(...any) error }) (*Homework, error) {
    var h Homework
    err := row.Scan(&h.ID, &h.ClassID, &h.ClassName, &h.SubjectID, &h.SubjectName, &h.Title, &h.Details,
        &h.DueDate, &h.PostedBy, &h.PosterName)
    if err != nil {
        return nil, err
    }
    return &h, nil
}

func CreateHomework(db *sql.DB, in HomeworkInput) error {
    _, err := db.Exec(`INSERT INTO homework (class_id, subject_id, title, details, due_date, posted_by)
        VALUES (?, ?, ?, ?, ?, ?)`, in.ClassID, in.SubjectID, in.Title, in.Details, in.DueDate, nullInt(in.PostedBy))
    return err
}

func GetHomework(db *sql.DB, id int) (*Homework, error) {
    h, err := scanHomework(db.QueryRow(homeworkSelect+" WHERE h.id = ?", id))
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    return h, err
}

// UpdateHomework changes the wording and due date. The class and subject stay as they were.
func UpdateHomework(db *sql.DB, id int, title, details, dueDate string) error {
    _, err := db.Exec("UPDATE homework SET title = ?, details = ?, due_date = ? WHERE id = ?",
        title, details, dueDate, id)
    return err
}

func DeleteHomework(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM homework WHERE id = ?", id)
    return err
}

// ListHomework returns homework due on or after sinceDate. classIDs limits it to those
// classes; nil means every class. The nearest due date comes first.
func ListHomework(db *sql.DB, classIDs []int, sinceDate string) ([]Homework, error) {
    query := homeworkSelect + " WHERE h.due_date >= ?"
    args := []any{sinceDate}
    if classIDs != nil {
        if len(classIDs) == 0 {
            return nil, nil
        }
        marks := strings.TrimSuffix(strings.Repeat("?,", len(classIDs)), ",")
        query += " AND h.class_id IN (" + marks + ")"
        for _, id := range classIDs {
            args = append(args, id)
        }
    }
    query += " ORDER BY h.due_date, c.name, b.name, h.id"

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Homework
    for rows.Next() {
        h, err := scanHomework(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *h)
    }
    return list, rows.Err()
}
