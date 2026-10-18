package models

import "database/sql"

const (
    StatusPresent = "present"
    StatusLate    = "late"
    StatusAbsent  = "absent"
)

func ValidStatus(s string) bool {
    return s == StatusPresent || s == StatusLate || s == StatusAbsent
}

type AttendanceRecord struct {
    Date   string
    Status string
}

type AttendanceSummary struct {
    Present int
    Late    int
    Absent  int
    Total   int
    Percent float64
}

type ClassReportRow struct {
    Roll    int
    Name    string
    Present int
    Late    int
    Absent  int
    Total   int
    Percent float64
}

// attendancePercent counts "late" as attended.
func attendancePercent(present, late, total int) float64 {
    if total == 0 {
        return 0
    }
    return float64(present+late) * 100 / float64(total)
}

// ClassesForUser returns every class for an admin, and only the classes where
// the teacher has an assigned subject for a teacher.
func ClassesForUser(db *sql.DB, role string, userID int) ([]Class, error) {
    if role == "admin" {
        return ListClasses(db)
    }
    rows, err := db.Query(`SELECT c.id, c.name FROM classes c
        WHERE c.id IN (
            SELECT b.class_id FROM teacher_subjects ts
            JOIN subjects b ON b.id = ts.subject_id
            JOIN teachers t ON t.id = ts.teacher_id
            WHERE t.user_id = ?)
        ORDER BY c.id`, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Class
    for rows.Next() {
        var c Class
        if err := rows.Scan(&c.ID, &c.Name); err != nil {
            return nil, err
        }
        list = append(list, c)
    }
    return list, rows.Err()
}

// ListClassStudents lists the students of a class, optionally limited to one section (0 = all).
func ListClassStudents(db *sql.DB, classID, sectionID int) ([]Student, error) {
    rows, err := db.Query(studentSelect+
        " WHERE s.class_id = ? AND (? = 0 OR s.section_id = ?) ORDER BY s.roll",
        classID, sectionID, sectionID)
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

// GetAttendanceMap returns the saved status of each student (by student ID) for one date.
func GetAttendanceMap(db *sql.DB, classID, sectionID int, date string) (map[int]string, error) {
    rows, err := db.Query(`SELECT a.student_id, a.status FROM attendance a
        JOIN students s ON s.id = a.student_id
        WHERE s.class_id = ? AND (? = 0 OR s.section_id = ?) AND a.date = ?`,
        classID, sectionID, sectionID, date)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    marks := make(map[int]string)
    for rows.Next() {
        var id int
        var status string
        if err := rows.Scan(&id, &status); err != nil {
            return nil, err
        }
        marks[id] = status
    }
    return marks, rows.Err()
}

// SaveAttendance saves all marks in one transaction. Saving a date again updates the old marks.
func SaveAttendance(db *sql.DB, date string, marks map[int]string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for studentID, status := range marks {
        _, err := tx.Exec(`INSERT INTO attendance (student_id, date, status) VALUES (?, ?, ?)
            ON CONFLICT(student_id, date) DO UPDATE SET status = excluded.status`,
            studentID, date, status)
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}

func GetStudentAttendance(db *sql.DB, studentID, limit int) ([]AttendanceRecord, error) {
    rows, err := db.Query(`SELECT date, status FROM attendance
        WHERE student_id = ? ORDER BY date DESC LIMIT ?`, studentID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []AttendanceRecord
    for rows.Next() {
        var rec AttendanceRecord
        if err := rows.Scan(&rec.Date, &rec.Status); err != nil {
            return nil, err
        }
        list = append(list, rec)
    }
    return list, rows.Err()
}

func GetStudentSummary(db *sql.DB, studentID int) (AttendanceSummary, error) {
    var s AttendanceSummary
    err := db.QueryRow(`SELECT
        COALESCE(SUM(status = 'present'), 0),
        COALESCE(SUM(status = 'late'), 0),
        COALESCE(SUM(status = 'absent'), 0),
        COUNT(*)
        FROM attendance WHERE student_id = ?`, studentID).
        Scan(&s.Present, &s.Late, &s.Absent, &s.Total)
    s.Percent = attendancePercent(s.Present, s.Late, s.Total)
    return s, err
}

// ClassReport counts each student's attendance between two dates (inclusive).
func ClassReport(db *sql.DB, classID, sectionID int, from, to string) ([]ClassReportRow, error) {
    rows, err := db.Query(`SELECT s.roll, s.name,
        COALESCE(SUM(a.status = 'present'), 0),
        COALESCE(SUM(a.status = 'late'), 0),
        COALESCE(SUM(a.status = 'absent'), 0),
        COUNT(a.id)
        FROM students s
        LEFT JOIN attendance a ON a.student_id = s.id AND a.date BETWEEN ? AND ?
        WHERE s.class_id = ? AND (? = 0 OR s.section_id = ?)
        GROUP BY s.id
        ORDER BY s.roll`, from, to, classID, sectionID, sectionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []ClassReportRow
    for rows.Next() {
        var r ClassReportRow
        if err := rows.Scan(&r.Roll, &r.Name, &r.Present, &r.Late, &r.Absent, &r.Total); err != nil {
            return nil, err
        }
        r.Percent = attendancePercent(r.Present, r.Late, r.Total)
        list = append(list, r)
    }
    return list, rows.Err()
}
