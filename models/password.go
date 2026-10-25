package models

import "database/sql"

type UserRow struct {
    ID       int
    Username string
    Role     string
    Name     string
}

func GetUserByID(db *sql.DB, id int) (*User, error) {
    u := &User{}
    err := db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE id = ?", id).
        Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
    if err != nil {
        return nil, notFoundIf(err)
    }
    return u, nil
}

// SetPassword hashes the new password with bcrypt and saves it.
func SetPassword(db *sql.DB, userID int, password string) error {
    hash, err := hashPassword(password)
    if err != nil {
        return err
    }
    res, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, userID)
    if err != nil {
        return err
    }
    n, err := res.RowsAffected()
    if err != nil {
        return err
    }
    if n == 0 {
        return ErrNotFound
    }
    return nil
}

// ListUsers lists every account with the name from its student, teacher or parent profile.
func ListUsers(db *sql.DB) ([]UserRow, error) {
    rows, err := db.Query(`SELECT u.id, u.username, u.role, COALESCE(s.name, t.name, p.name, '')
        FROM users u
        LEFT JOIN students s ON s.user_id = u.id
        LEFT JOIN teachers t ON t.user_id = u.id
        LEFT JOIN parents p ON p.user_id = u.id
        ORDER BY CASE u.role WHEN 'admin' THEN 0 WHEN 'teacher' THEN 1 WHEN 'parent' THEN 2 ELSE 3 END,
            u.username`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []UserRow
    for rows.Next() {
        var u UserRow
        if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Name); err != nil {
            return nil, err
        }
        list = append(list, u)
    }
    return list, rows.Err()
}
