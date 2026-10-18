package models

import "database/sql"

type Parent struct {
    ID         int
    UserID     int
    Username   string
    Name       string
    Phone      string
    ChildCount int
}

type ParentInput struct {
    Name     string
    Phone    string
    Username string
    Password string
}

// ChildOption is a student shown with a checkbox on the parent form.
type ChildOption struct {
    ID          int
    Name        string
    Roll        int
    ClassName   string
    SectionName string
    Linked      bool
}

const parentSelect = `SELECT p.id, p.user_id, u.username, p.name, p.phone,
    (SELECT COUNT(*) FROM parent_students ps WHERE ps.parent_id = p.id)
    FROM parents p
    JOIN users u ON u.id = p.user_id`

func scanParent(row rowScanner) (*Parent, error) {
    p := &Parent{}
    err := row.Scan(&p.ID, &p.UserID, &p.Username, &p.Name, &p.Phone, &p.ChildCount)
    if err != nil {
        return nil, err
    }
    return p, nil
}

func ListParents(db *sql.DB) ([]Parent, error) {
    rows, err := db.Query(parentSelect + " ORDER BY p.name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Parent
    for rows.Next() {
        p, err := scanParent(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *p)
    }
    return list, rows.Err()
}

func GetParent(db *sql.DB, id int) (*Parent, error) {
    p, err := scanParent(db.QueryRow(parentSelect+" WHERE p.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return p, nil
}

func GetParentByUserID(db *sql.DB, userID int) (*Parent, error) {
    p, err := scanParent(db.QueryRow(parentSelect+" WHERE p.user_id = ?", userID))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return p, nil
}

// CreateParent creates the login account and the parent profile in one transaction.
func CreateParent(db *sql.DB, in ParentInput) error {
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
        "INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'parent')",
        in.Username, hash)
    if err != nil {
        return err
    }
    userID, err := res.LastInsertId()
    if err != nil {
        return err
    }

    _, err = tx.Exec("INSERT INTO parents (user_id, name, phone) VALUES (?, ?, ?)",
        userID, in.Name, in.Phone)
    if err != nil {
        return err
    }
    return tx.Commit()
}

func UpdateParent(db *sql.DB, id int, in ParentInput) error {
    _, err := db.Exec("UPDATE parents SET name = ?, phone = ? WHERE id = ?", in.Name, in.Phone, id)
    return err
}

// DeleteParent removes the login account; the parent profile and child links go with it.
func DeleteParent(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM users WHERE id = (SELECT user_id FROM parents WHERE id = ?)", id)
    return err
}

// ListChildOptions lists every student, marking the ones linked to the parent.
func ListChildOptions(db *sql.DB, parentID int) ([]ChildOption, error) {
    rows, err := db.Query(`SELECT s.id, s.name, s.roll, COALESCE(c.name, ''), COALESCE(sec.name, ''),
        EXISTS (SELECT 1 FROM parent_students ps WHERE ps.student_id = s.id AND ps.parent_id = ?)
        FROM students s
        LEFT JOIN classes c ON c.id = s.class_id
        LEFT JOIN sections sec ON sec.id = s.section_id
        ORDER BY COALESCE(s.class_id, 0), s.roll`, parentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []ChildOption
    for rows.Next() {
        var o ChildOption
        var linked int
        if err := rows.Scan(&o.ID, &o.Name, &o.Roll, &o.ClassName, &o.SectionName, &linked); err != nil {
            return nil, err
        }
        o.Linked = linked == 1
        list = append(list, o)
    }
    return list, rows.Err()
}

// ListChildren lists only the students linked to the parent.
func ListChildren(db *sql.DB, parentID int) ([]Student, error) {
    rows, err := db.Query(studentSelect+
        " JOIN parent_students ps ON ps.student_id = s.id WHERE ps.parent_id = ? ORDER BY s.name", parentID)
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

// ParentHasChild reports whether the student is linked to the parent.
func ParentHasChild(db *sql.DB, parentID, studentID int) (bool, error) {
    var n int
    err := db.QueryRow("SELECT COUNT(*) FROM parent_students WHERE parent_id = ? AND student_id = ?",
        parentID, studentID).Scan(&n)
    return n > 0, err
}

// SetParentChildren replaces the parent's list of children. Unknown student IDs are ignored.
func SetParentChildren(db *sql.DB, parentID int, studentIDs []int) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.Exec("DELETE FROM parent_students WHERE parent_id = ?", parentID); err != nil {
        return err
    }
    for _, sid := range studentIDs {
        _, err := tx.Exec(`INSERT OR IGNORE INTO parent_students (parent_id, student_id)
            SELECT ?, id FROM students WHERE id = ?`, parentID, sid)
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}
