package models

import "database/sql"

type Teacher struct {
    ID            int
    UserID        int
    Username      string
    Name          string
    Phone         string
    Qualification string
    SubjectCount  int
}

type TeacherInput struct {
    Name          string
    Phone         string
    Qualification string
    Username      string
    Password      string
}

// SubjectOption is a subject shown with its class, and whether a teacher is assigned to it.
type SubjectOption struct {
    ID        int
    Name      string
    ClassName string
    Assigned  bool
}

const teacherSelect = `SELECT t.id, t.user_id, u.username, t.name, t.phone, t.qualification,
    (SELECT COUNT(*) FROM teacher_subjects ts WHERE ts.teacher_id = t.id)
    FROM teachers t
    JOIN users u ON u.id = t.user_id`

func scanTeacher(row rowScanner) (*Teacher, error) {
    t := &Teacher{}
    err := row.Scan(&t.ID, &t.UserID, &t.Username, &t.Name, &t.Phone, &t.Qualification, &t.SubjectCount)
    if err != nil {
        return nil, err
    }
    return t, nil
}

func ListTeachers(db *sql.DB) ([]Teacher, error) {
    rows, err := db.Query(teacherSelect + " ORDER BY t.name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Teacher
    for rows.Next() {
        t, err := scanTeacher(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *t)
    }
    return list, rows.Err()
}

func GetTeacher(db *sql.DB, id int) (*Teacher, error) {
    t, err := scanTeacher(db.QueryRow(teacherSelect+" WHERE t.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return t, nil
}

func GetTeacherByUserID(db *sql.DB, userID int) (*Teacher, error) {
    t, err := scanTeacher(db.QueryRow(teacherSelect+" WHERE t.user_id = ?", userID))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return t, nil
}

// CreateTeacher creates the login account and the teacher profile in one transaction.
func CreateTeacher(db *sql.DB, in TeacherInput) error {
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
        "INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'teacher')",
        in.Username, hash)
    if err != nil {
        return err
    }
    userID, err := res.LastInsertId()
    if err != nil {
        return err
    }

    _, err = tx.Exec("INSERT INTO teachers (user_id, name, phone, qualification) VALUES (?, ?, ?, ?)",
        userID, in.Name, in.Phone, in.Qualification)
    if err != nil {
        return err
    }
    return tx.Commit()
}

func UpdateTeacher(db *sql.DB, id int, in TeacherInput) error {
    _, err := db.Exec("UPDATE teachers SET name = ?, phone = ?, qualification = ? WHERE id = ?",
        in.Name, in.Phone, in.Qualification, id)
    return err
}

// DeleteTeacher removes the login account; the teacher profile and subject links go with it.
func DeleteTeacher(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM users WHERE id = (SELECT user_id FROM teachers WHERE id = ?)", id)
    return err
}

// ListSubjectOptions lists every subject, marking the ones assigned to the teacher.
func ListSubjectOptions(db *sql.DB, teacherID int) ([]SubjectOption, error) {
    rows, err := db.Query(`SELECT b.id, b.name, c.name,
        EXISTS (SELECT 1 FROM teacher_subjects ts WHERE ts.subject_id = b.id AND ts.teacher_id = ?)
        FROM subjects b
        JOIN classes c ON c.id = b.class_id
        ORDER BY c.id, b.name`, teacherID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []SubjectOption
    for rows.Next() {
        var o SubjectOption
        var assigned int
        if err := rows.Scan(&o.ID, &o.Name, &o.ClassName, &assigned); err != nil {
            return nil, err
        }
        o.Assigned = assigned == 1
        list = append(list, o)
    }
    return list, rows.Err()
}

// ListAssignedSubjects lists only the subjects assigned to the teacher.
func ListAssignedSubjects(db *sql.DB, teacherID int) ([]SubjectOption, error) {
    rows, err := db.Query(`SELECT b.id, b.name, c.name
        FROM teacher_subjects ts
        JOIN subjects b ON b.id = ts.subject_id
        JOIN classes c ON c.id = b.class_id
        WHERE ts.teacher_id = ?
        ORDER BY c.id, b.name`, teacherID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []SubjectOption
    for rows.Next() {
        o := SubjectOption{Assigned: true}
        if err := rows.Scan(&o.ID, &o.Name, &o.ClassName); err != nil {
            return nil, err
        }
        list = append(list, o)
    }
    return list, rows.Err()
}

// SetTeacherSubjects replaces the teacher's subject list. Unknown subject IDs are ignored.
func SetTeacherSubjects(db *sql.DB, teacherID int, subjectIDs []int) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.Exec("DELETE FROM teacher_subjects WHERE teacher_id = ?", teacherID); err != nil {
        return err
    }
    for _, sid := range subjectIDs {
        _, err := tx.Exec(`INSERT OR IGNORE INTO teacher_subjects (teacher_id, subject_id)
            SELECT ?, id FROM subjects WHERE id = ?`, teacherID, sid)
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}
