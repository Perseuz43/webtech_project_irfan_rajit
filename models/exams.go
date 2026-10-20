package models

import "database/sql"

type Exam struct {
    ID        int
    ClassID   int
    ClassName string
    Name      string
    Date      string
    Published bool
}

type ExamSubject struct {
    SubjectID int
    Name      string
    FullMarks float64
}

// MarkEntry is one student's mark on the entry sheet. Has is false when the box was left empty.
type MarkEntry struct {
    StudentID int
    Marks     float64
    Has       bool
}

const examSelect = `SELECT e.id, e.class_id, c.name, e.name, e.exam_date, e.published
    FROM exams e JOIN classes c ON c.id = e.class_id`

const examOrder = " ORDER BY e.exam_date DESC, e.id DESC"

const syncSQL = `INSERT OR IGNORE INTO exam_subjects (exam_id, subject_id)
    SELECT e.id, b.id FROM exams e JOIN subjects b ON b.class_id = e.class_id WHERE e.id = ?`

func scanExam(row rowScanner) (*Exam, error) {
    e := &Exam{}
    var published int
    if err := row.Scan(&e.ID, &e.ClassID, &e.ClassName, &e.Name, &e.Date, &published); err != nil {
        return nil, err
    }
    e.Published = published == 1
    return e, nil
}

func queryExams(db *sql.DB, query string, args ...any) ([]Exam, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Exam
    for rows.Next() {
        e, err := scanExam(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *e)
    }
    return list, rows.Err()
}

func ListExams(db *sql.DB) ([]Exam, error) {
    return queryExams(db, examSelect+examOrder)
}

// ListExamsForUser returns every exam for an admin, and only the exams of classes
// where the teacher has an assigned subject.
func ListExamsForUser(db *sql.DB, role string, userID int) ([]Exam, error) {
    if role == "admin" {
        return ListExams(db)
    }
    return queryExams(db, examSelect+` WHERE e.class_id IN (
        SELECT b.class_id FROM teacher_subjects ts
        JOIN subjects b ON b.id = ts.subject_id
        JOIN teachers t ON t.id = ts.teacher_id
        WHERE t.user_id = ?)`+examOrder, userID)
}

func ListPublishedExamsForClass(db *sql.DB, classID int) ([]Exam, error) {
    return queryExams(db, examSelect+" WHERE e.class_id = ? AND e.published = 1"+examOrder, classID)
}

func GetExam(db *sql.DB, id int) (*Exam, error) {
    e, err := scanExam(db.QueryRow(examSelect+" WHERE e.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return e, nil
}

// CreateExam creates the exam and gives it every subject of the class (100 full marks each).
func CreateExam(db *sql.DB, classID int, name, date string) (int, error) {
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()

    res, err := tx.Exec("INSERT INTO exams (class_id, name, exam_date) VALUES (?, ?, ?)", classID, name, date)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }
    if _, err := tx.Exec(syncSQL, id); err != nil {
        return 0, err
    }
    return int(id), tx.Commit()
}

// SyncExamSubjects adds any subject the class gained after the exam was created.
func SyncExamSubjects(db *sql.DB, examID int) error {
    _, err := db.Exec(syncSQL, examID)
    return err
}

func DeleteExam(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM exams WHERE id = ?", id)
    return err
}

func SetExamPublished(db *sql.DB, id int, published bool) error {
    v := 0
    if published {
        v = 1
    }
    _, err := db.Exec("UPDATE exams SET published = ? WHERE id = ?", v, id)
    return err
}

func queryExamSubjects(db *sql.DB, query string, args ...any) ([]ExamSubject, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []ExamSubject
    for rows.Next() {
        var s ExamSubject
        if err := rows.Scan(&s.SubjectID, &s.Name, &s.FullMarks); err != nil {
            return nil, err
        }
        list = append(list, s)
    }
    return list, rows.Err()
}

func ListExamSubjects(db *sql.DB, examID int) ([]ExamSubject, error) {
    return queryExamSubjects(db, `SELECT es.subject_id, b.name, es.full_marks
        FROM exam_subjects es JOIN subjects b ON b.id = es.subject_id
        WHERE es.exam_id = ? ORDER BY b.name`, examID)
}

// ListMarkableSubjects lists the exam subjects the user may enter marks for:
// all of them for an admin, only the assigned ones for a teacher.
func ListMarkableSubjects(db *sql.DB, role string, userID, examID int) ([]ExamSubject, error) {
    if role == "admin" {
        return ListExamSubjects(db, examID)
    }
    return queryExamSubjects(db, `SELECT es.subject_id, b.name, es.full_marks
        FROM exam_subjects es JOIN subjects b ON b.id = es.subject_id
        WHERE es.exam_id = ? AND es.subject_id IN (
            SELECT ts.subject_id FROM teacher_subjects ts
            JOIN teachers t ON t.id = ts.teacher_id WHERE t.user_id = ?)
        ORDER BY b.name`, examID, userID)
}

// SaveFullMarks updates the full marks. If a new value is lower than a mark that is
// already entered, nothing is saved and the subject's name is returned.
func SaveFullMarks(db *sql.DB, examID int, full map[int]float64) (string, error) {
    tx, err := db.Begin()
    if err != nil {
        return "", err
    }
    defer tx.Rollback()

    for subjectID, v := range full {
        var n int
        err := tx.QueryRow("SELECT COUNT(*) FROM marks WHERE exam_id = ? AND subject_id = ? AND marks > ?",
            examID, subjectID, v).Scan(&n)
        if err != nil {
            return "", err
        }
        if n > 0 {
            var name string
            if err := tx.QueryRow("SELECT name FROM subjects WHERE id = ?", subjectID).Scan(&name); err != nil {
                return "", err
            }
            return name, nil
        }
        _, err = tx.Exec("UPDATE exam_subjects SET full_marks = ? WHERE exam_id = ? AND subject_id = ?",
            v, examID, subjectID)
        if err != nil {
            return "", err
        }
    }
    return "", tx.Commit()
}

// GetMarksMap returns the saved marks of one subject, keyed by student ID.
func GetMarksMap(db *sql.DB, examID, subjectID int) (map[int]float64, error) {
    rows, err := db.Query("SELECT student_id, marks FROM marks WHERE exam_id = ? AND subject_id = ?", examID, subjectID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    m := make(map[int]float64)
    for rows.Next() {
        var id int
        var v float64
        if err := rows.Scan(&id, &v); err != nil {
            return nil, err
        }
        m[id] = v
    }
    return m, rows.Err()
}

// SaveMarks saves the whole sheet in one transaction. An empty box removes the saved mark.
func SaveMarks(db *sql.DB, examID, subjectID int, entries []MarkEntry) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, e := range entries {
        if e.Has {
            _, err = tx.Exec(`INSERT INTO marks (exam_id, subject_id, student_id, marks) VALUES (?, ?, ?, ?)
                ON CONFLICT(exam_id, subject_id, student_id) DO UPDATE SET marks = excluded.marks`,
                examID, subjectID, e.StudentID, e.Marks)
        } else {
            _, err = tx.Exec("DELETE FROM marks WHERE exam_id = ? AND subject_id = ? AND student_id = ?",
                examID, subjectID, e.StudentID)
        }
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}
