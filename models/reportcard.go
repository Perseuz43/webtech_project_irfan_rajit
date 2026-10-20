package models

import "database/sql"

type SubjectResult struct {
    Subject  string
    Full     float64
    Marks    float64
    HasMarks bool
    Percent  float64
    Grade    string
    Point    float64
    Passed   bool
}

type ReportCard struct {
    Student    *Student
    Exam       *Exam
    Scale      *GradingScale
    Rows       []SubjectResult
    TotalMarks float64
    TotalFull  float64
    Percent    float64
    Grade      string
    GPA        float64
    HasGPA     bool
    Status     string // Passed, Failed or Incomplete
}

// BuildReportCard works out grades, totals, GPA and the result using the given grading system.
func BuildReportCard(db *sql.DB, student *Student, exam *Exam, scale *GradingScale) (*ReportCard, error) {
    rows, err := db.Query(`SELECT b.name, es.full_marks, m.marks
        FROM exam_subjects es
        JOIN subjects b ON b.id = es.subject_id
        LEFT JOIN marks m ON m.exam_id = es.exam_id AND m.subject_id = es.subject_id AND m.student_id = ?
        WHERE es.exam_id = ?
        ORDER BY b.name`, student.ID, exam.ID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    rc := &ReportCard{Student: student, Exam: exam, Scale: scale}
    var points float64
    entered, failed, missing := 0, 0, 0

    for rows.Next() {
        var r SubjectResult
        var m sql.NullFloat64
        if err := rows.Scan(&r.Subject, &r.Full, &m); err != nil {
            return nil, err
        }
        if m.Valid {
            r.HasMarks = true
            r.Marks = m.Float64
            r.Percent = r.Marks * 100 / r.Full
            r.Grade, r.Point = scale.Grade(r.Percent)
            r.Passed = scale.Passed(r.Percent)

            rc.TotalMarks += r.Marks
            rc.TotalFull += r.Full
            points += r.Point
            entered++
            if !r.Passed {
                failed++
            }
        } else {
            missing++
        }
        rc.Rows = append(rc.Rows, r)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }

    rc.Grade = "-"
    if rc.TotalFull > 0 {
        rc.Percent = rc.TotalMarks * 100 / rc.TotalFull
        rc.Grade, _ = scale.Grade(rc.Percent)
    }

    switch {
    case len(rc.Rows) == 0 || missing > 0:
        rc.Status = "Incomplete"
    case failed > 0:
        rc.Status = "Failed"
    default:
        rc.Status = "Passed"
    }

    if scale.UseGPA && rc.Status != "Incomplete" && entered > 0 {
        rc.GPA = points / float64(entered)
        rc.HasGPA = true
    }
    return rc, nil
}
