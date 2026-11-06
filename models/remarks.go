package models

import (
    "database/sql"
    "errors"
)

// MaxRemarkLength is the longest teacher's remark, in characters.
const MaxRemarkLength = 300

// GetRemark returns the remark for one student in one exam ("" when there is none).
func GetRemark(db *sql.DB, examID, studentID int) (string, error) {
    var remark string
    err := db.QueryRow("SELECT remark FROM report_remarks WHERE exam_id = ? AND student_id = ?",
        examID, studentID).Scan(&remark)
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil
    }
    return remark, err
}

// GetRemarksMap returns every remark of an exam, keyed by student ID.
func GetRemarksMap(db *sql.DB, examID int) (map[int]string, error) {
    rows, err := db.Query("SELECT student_id, remark FROM report_remarks WHERE exam_id = ?", examID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    m := make(map[int]string)
    for rows.Next() {
        var id int
        var remark string
        if err := rows.Scan(&id, &remark); err != nil {
            return nil, err
        }
        m[id] = remark
    }
    return m, rows.Err()
}

// SaveRemarks stores the remarks of an exam in one go. An empty remark removes the old one.
func SaveRemarks(db *sql.DB, examID int, remarks map[int]string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for studentID, remark := range remarks {
        if remark == "" {
            _, err = tx.Exec("DELETE FROM report_remarks WHERE exam_id = ? AND student_id = ?", examID, studentID)
        } else {
            _, err = tx.Exec(`INSERT INTO report_remarks (exam_id, student_id, remark) VALUES (?, ?, ?)
                ON CONFLICT(exam_id, student_id) DO UPDATE SET remark = excluded.remark`,
                examID, studentID, remark)
        }
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}
