package models

import (
    "database/sql"
    "strconv"
    "strings"
)

type ParentContact struct {
    Name  string
    Phone string
}

type DirectoryEntry struct {
    StudentID     int
    Roll          int
    Name          string
    ClassName     string
    SectionName   string
    GuardianName  string // from the student's own profile
    GuardianPhone string
    Parents       []ParentContact // from linked parent accounts
}

func escapeLike(s string) string {
    return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// directoryFilter builds the WHERE conditions. Anyone who is not an admin only gets students
// from classes where they have an assigned subject, so a wrong role can never see more.
func directoryFilter(role string, userID, classID, sectionID int, query string) (string, []any) {
    conds := []string{"1 = 1"}
    var args []any

    if role != "admin" {
        conds = append(conds, `s.class_id IN (
            SELECT b.class_id FROM teacher_subjects ts
            JOIN subjects b ON b.id = ts.subject_id
            JOIN teachers t ON t.id = ts.teacher_id
            WHERE t.user_id = ?)`)
        args = append(args, userID)
    }
    if classID > 0 {
        conds = append(conds, "s.class_id = ?")
        args = append(args, classID)
    }
    if sectionID > 0 {
        conds = append(conds, "s.section_id = ?")
        args = append(args, sectionID)
    }
    if query != "" {
        like := "%" + escapeLike(query) + "%"
        if n, err := strconv.Atoi(query); err == nil {
            conds = append(conds, `(s.name LIKE ? ESCAPE '\' OR s.roll = ?)`)
            args = append(args, like, n)
        } else {
            conds = append(conds, `s.name LIKE ? ESCAPE '\'`)
            args = append(args, like)
        }
    }
    return strings.Join(conds, " AND "), args
}

// ListDirectory lists students with their class and parent contacts, ordered by class, section and roll.
// classID and sectionID of 0 mean "all".
func ListDirectory(db *sql.DB, role string, userID, classID, sectionID int, query string) ([]DirectoryEntry, error) {
    where, args := directoryFilter(role, userID, classID, sectionID, query)

    rows, err := db.Query(`SELECT s.id, s.roll, s.name, COALESCE(c.name, ''), COALESCE(sec.name, ''),
        s.guardian_name, s.phone
        FROM students s
        LEFT JOIN classes c ON c.id = s.class_id
        LEFT JOIN sections sec ON sec.id = s.section_id
        WHERE `+where+`
        ORDER BY COALESCE(s.class_id, 0), COALESCE(sec.name, ''), s.roll`, args...)
    if err != nil {
        return nil, err
    }

    var list []DirectoryEntry
    index := make(map[int]int)
    for rows.Next() {
        var e DirectoryEntry
        if err := rows.Scan(&e.StudentID, &e.Roll, &e.Name, &e.ClassName, &e.SectionName,
            &e.GuardianName, &e.GuardianPhone); err != nil {
            rows.Close()
            return nil, err
        }
        index[e.StudentID] = len(list)
        list = append(list, e)
    }
    if err := rows.Err(); err != nil {
        rows.Close()
        return nil, err
    }
    rows.Close()
    if len(list) == 0 {
        return list, nil
    }

    prows, err := db.Query(`SELECT ps.student_id, p.name, p.phone
        FROM parent_students ps
        JOIN parents p ON p.id = ps.parent_id
        WHERE ps.student_id IN (SELECT s.id FROM students s WHERE `+where+`)
        ORDER BY p.name`, args...)
    if err != nil {
        return nil, err
    }
    defer prows.Close()

    for prows.Next() {
        var sid int
        var pc ParentContact
        if err := prows.Scan(&sid, &pc.Name, &pc.Phone); err != nil {
            return nil, err
        }
        if i, ok := index[sid]; ok {
            list[i].Parents = append(list[i].Parents, pc)
        }
    }
    return list, prows.Err()
}
