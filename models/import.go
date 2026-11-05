package models

import (
    "database/sql"
    "runtime"
    "strings"
    "sync"
)

// ExistingUsernames returns every username that is already taken.
func ExistingUsernames(db *sql.DB) (map[string]bool, error) {
    rows, err := db.Query("SELECT username FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    taken := make(map[string]bool)
    for rows.Next() {
        var u string
        if err := rows.Scan(&u); err != nil {
            return nil, err
        }
        taken[u] = true
    }
    return taken, rows.Err()
}

// RollKey identifies "roll 5 in class 3" so duplicates can be spotted.
type RollKey struct {
    ClassID int
    Roll    int
}

// ExistingRolls returns every (class, roll) pair that is already used.
func ExistingRolls(db *sql.DB) (map[RollKey]bool, error) {
    rows, err := db.Query("SELECT class_id, roll FROM students WHERE class_id IS NOT NULL")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    used := make(map[RollKey]bool)
    for rows.Next() {
        var k RollKey
        if err := rows.Scan(&k.ClassID, &k.Roll); err != nil {
            return nil, err
        }
        used[k] = true
    }
    return used, rows.Err()
}

// ClassNamesToIDs maps a lower-case class name to its ID.
func ClassNamesToIDs(db *sql.DB) (map[string]int, error) {
    rows, err := db.Query("SELECT id, name FROM classes")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    m := make(map[string]int)
    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            return nil, err
        }
        m[strings.ToLower(strings.TrimSpace(name))] = id
    }
    return m, rows.Err()
}

// ImportStudents creates every student in one transaction: either all of them are
// saved or none are. Passwords are hashed in parallel first, because bcrypt is slow
// on purpose and 200 students would otherwise take several seconds.
func ImportStudents(db *sql.DB, list []StudentInput) error {
    hashes := make([]string, len(list))

    var wg sync.WaitGroup
    var mu sync.Mutex
    var firstErr error
    slots := make(chan struct{}, runtime.NumCPU())

    for i := range list {
        wg.Add(1)
        slots <- struct{}{}
        go func(i int) {
            defer wg.Done()
            defer func() { <-slots }()
            h, err := hashPassword(list[i].Password)
            if err != nil {
                mu.Lock()
                if firstErr == nil {
                    firstErr = err
                }
                mu.Unlock()
                return
            }
            hashes[i] = h
        }(i)
    }
    wg.Wait()
    if firstErr != nil {
        return firstErr
    }

    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for i, in := range list {
        res, err := tx.Exec(
            "INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'student')",
            in.Username, hashes[i])
        if err != nil {
            return err
        }
        userID, err := res.LastInsertId()
        if err != nil {
            return err
        }
        _, err = tx.Exec(`INSERT INTO students (user_id, name, roll, class_id, section_id, guardian_name, phone)
            VALUES (?, ?, ?, ?, ?, ?, ?)`,
            userID, in.Name, in.Roll, nullInt(in.ClassID), nullInt(in.SectionID), in.GuardianName, in.Phone)
        if err != nil {
            return err
        }
    }
    return tx.Commit()
}
