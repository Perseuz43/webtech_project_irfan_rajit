package models

import (
    "database/sql"
    "errors"
    "strconv"
    "time"
)

// SetStudentPhoto saves (or replaces) a student's photo.
func SetStudentPhoto(db *sql.DB, studentID int, contentType string, data []byte) error {
    _, err := db.Exec(`INSERT INTO student_photos (student_id, content_type, data, version) VALUES (?, ?, ?, ?)
        ON CONFLICT(student_id) DO UPDATE SET content_type = excluded.content_type,
            data = excluded.data, version = excluded.version`,
        studentID, contentType, data, strconv.FormatInt(time.Now().UnixNano(), 10))
    return err
}

func GetStudentPhoto(db *sql.DB, studentID int) (string, []byte, error) {
    var contentType string
    var data []byte
    err := db.QueryRow("SELECT content_type, data FROM student_photos WHERE student_id = ?", studentID).
        Scan(&contentType, &data)
    if err != nil {
        return "", nil, notFoundIf(err)
    }
    return contentType, data, nil
}

func DeleteStudentPhoto(db *sql.DB, studentID int) error {
    _, err := db.Exec("DELETE FROM student_photos WHERE student_id = ?", studentID)
    return err
}

// StudentPhotoVersion returns the photo's version string, or "" when the student has no photo.
// The version goes into the image address so browsers load the new picture after a change.
func StudentPhotoVersion(db *sql.DB, studentID int) (string, error) {
    var v string
    err := db.QueryRow("SELECT version FROM student_photos WHERE student_id = ?", studentID).Scan(&v)
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil
    }
    return v, err
}

// PhotoVersions returns the version of every stored photo, keyed by student ID.
func PhotoVersions(db *sql.DB) (map[int]string, error) {
    rows, err := db.Query("SELECT student_id, version FROM student_photos")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    m := make(map[int]string)
    for rows.Next() {
        var id int
        var v string
        if err := rows.Scan(&id, &v); err != nil {
            return nil, err
        }
        m[id] = v
    }
    return m, rows.Err()
}
