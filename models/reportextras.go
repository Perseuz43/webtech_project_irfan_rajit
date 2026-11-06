package models

import (
    "database/sql"
    "math"
    "strconv"
)

// Ordinal turns 1 into "1st", 2 into "2nd", 11 into "11th" and so on.
func Ordinal(n int) string {
    suffix := "th"
    if n%100 < 11 || n%100 > 13 {
        switch n % 10 {
        case 1:
            suffix = "st"
        case 2:
            suffix = "nd"
        case 3:
            suffix = "rd"
        }
    }
    return strconv.Itoa(n) + suffix
}

// RankText is "2nd of 18", or "" when the student has no position.
func (rc *ReportCard) RankText() string {
    if !rc.HasRank {
        return ""
    }
    return Ordinal(rc.Rank) + " of " + strconv.Itoa(rc.RankOf)
}

// ClassRank works out a student's position in an exam. Only students whose marks are
// complete (every subject entered) are ranked, so a missing mark never pushes someone
// up or down. Students with the same percentage share a position (1, 2, 2, 4).
// ok is false when this student is not ranked.
func ClassRank(db *sql.DB, examID, studentID int) (rank, of int, ok bool, err error) {
    rows, err := db.Query(`SELECT s.id,
            COALESCE(SUM(m.marks), 0), SUM(es.full_marks),
            SUM(CASE WHEN m.marks IS NULL THEN 1 ELSE 0 END)
        FROM exams e
        JOIN students s ON s.class_id = e.class_id
        JOIN exam_subjects es ON es.exam_id = e.id
        LEFT JOIN marks m ON m.exam_id = es.exam_id AND m.subject_id = es.subject_id AND m.student_id = s.id
        WHERE e.id = ?
        GROUP BY s.id`, examID)
    if err != nil {
        return 0, 0, false, err
    }
    defer rows.Close()

    var percents []float64
    var mine float64
    found := false
    for rows.Next() {
        var id, missing int
        var marks, full float64
        if err := rows.Scan(&id, &marks, &full, &missing); err != nil {
            return 0, 0, false, err
        }
        if missing > 0 || full <= 0 {
            continue
        }
        p := marks * 100 / full
        percents = append(percents, p)
        if id == studentID {
            mine, found = p, true
        }
    }
    if err := rows.Err(); err != nil {
        return 0, 0, false, err
    }
    if !found {
        return 0, 0, false, nil
    }
    rank = 1
    for _, p := range percents {
        if p > mine+1e-9 {
            rank++
        }
    }
    return rank, len(percents), true, nil
}

// AttendanceUpTo summarises a student's attendance on or before the given date (YYYY-MM-DD).
// "Late" counts as attended, the same as everywhere else in the app.
func AttendanceUpTo(db *sql.DB, studentID int, date string) (AttendanceSummary, error) {
    var s AttendanceSummary
    err := db.QueryRow(`SELECT
            COALESCE(SUM(status = 'present'), 0),
            COALESCE(SUM(status = 'late'), 0),
            COALESCE(SUM(status = 'absent'), 0)
        FROM attendance WHERE student_id = ? AND date <= ?`, studentID, date).
        Scan(&s.Present, &s.Late, &s.Absent)
    if err != nil {
        return s, err
    }
    s.Total = s.Present + s.Late + s.Absent
    s.Percent = math.Round(attendancePercent(s.Present, s.Late, s.Total)*10) / 10
    return s, nil
}
