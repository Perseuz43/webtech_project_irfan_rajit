package models

import (
    "database/sql"
    "strconv"
    "time"
)

type Notice struct {
    ID        int
    Title     string
    Body      string
    Audience  string
    ClassID   int
    ClassName string
    Pinned    bool
    CreatedAt string
}

type NoticeInput struct {
    Title    string
    Body     string
    Audience string // all, teacher, student, parent or class
    ClassID  int    // only for the class audience, otherwise 0
    Pinned   bool
}

// AudienceLabel is the text shown on the notice.
func (n Notice) AudienceLabel() string {
    switch n.Audience {
    case "teacher":
        return "Teachers"
    case "student":
        return "Students"
    case "parent":
        return "Parents"
    case "class":
        return "Class: " + n.ClassName
    }
    return "Everyone"
}

// AudienceValue is the value of the audience drop-down on the form ("class:3" for a class).
func (n Notice) AudienceValue() string {
    if n.Audience == "class" {
        return "class:" + strconv.Itoa(n.ClassID)
    }
    return n.Audience
}

const noticeSelect = `SELECT n.id, n.title, n.body, n.audience, COALESCE(n.class_id, 0),
    COALESCE(c.name, ''), n.pinned, n.created_at
    FROM notices n LEFT JOIN classes c ON c.id = n.class_id`

func scanNotice(row rowScanner) (*Notice, error) {
    n := &Notice{}
    var pinned int
    err := row.Scan(&n.ID, &n.Title, &n.Body, &n.Audience, &n.ClassID, &n.ClassName, &pinned, &n.CreatedAt)
    if err != nil {
        return nil, err
    }
    n.Pinned = pinned == 1
    return n, nil
}

func queryNotices(db *sql.DB, query string, args ...any) ([]Notice, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Notice
    for rows.Next() {
        n, err := scanNotice(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *n)
    }
    return list, rows.Err()
}

// ListNoticesForUser returns the notices this user should see, important ones first, newest first.
// Admins see every notice.
func ListNoticesForUser(db *sql.DB, role string, userID, limit int) ([]Notice, error) {
    var where string
    var args []any

    switch role {
    case "admin":
        where = "1 = 1"
    case "teacher":
        where = `(n.audience IN ('all', 'teacher') OR (n.audience = 'class' AND n.class_id IN (
            SELECT b.class_id FROM teacher_subjects ts
            JOIN subjects b ON b.id = ts.subject_id
            JOIN teachers t ON t.id = ts.teacher_id
            WHERE t.user_id = ?)))`
        args = append(args, userID)
    case "student":
        where = `(n.audience IN ('all', 'student') OR (n.audience = 'class' AND n.class_id IN (
            SELECT s.class_id FROM students s WHERE s.user_id = ? AND s.class_id IS NOT NULL)))`
        args = append(args, userID)
    case "parent":
        where = `(n.audience IN ('all', 'parent') OR (n.audience = 'class' AND n.class_id IN (
            SELECT s.class_id FROM parent_students ps
            JOIN students s ON s.id = ps.student_id
            JOIN parents p ON p.id = ps.parent_id
            WHERE p.user_id = ? AND s.class_id IS NOT NULL)))`
        args = append(args, userID)
    default:
        where = "n.audience = 'all'"
    }

    query := noticeSelect + " WHERE " + where +
        " ORDER BY n.pinned DESC, n.created_at DESC, n.id DESC LIMIT ?"
    args = append(args, limit)
    return queryNotices(db, query, args...)
}

func GetNotice(db *sql.DB, id int) (*Notice, error) {
    n, err := scanNotice(db.QueryRow(noticeSelect+" WHERE n.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return n, nil
}

func pinnedInt(b bool) int {
    if b {
        return 1
    }
    return 0
}

func CreateNotice(db *sql.DB, in NoticeInput) error {
    _, err := db.Exec(`INSERT INTO notices (title, body, audience, class_id, pinned, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
        in.Title, in.Body, in.Audience, nullInt(in.ClassID), pinnedInt(in.Pinned),
        time.Now().Format("2006-01-02 15:04"))
    return err
}

func UpdateNotice(db *sql.DB, id int, in NoticeInput) error {
    _, err := db.Exec(`UPDATE notices SET title = ?, body = ?, audience = ?, class_id = ?, pinned = ?
        WHERE id = ?`,
        in.Title, in.Body, in.Audience, nullInt(in.ClassID), pinnedInt(in.Pinned), id)
    return err
}

func DeleteNotice(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM notices WHERE id = ?", id)
    return err
}
