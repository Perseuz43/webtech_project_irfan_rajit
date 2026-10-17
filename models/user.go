package models

import (
    "database/sql"
    "errors"
    "log"

    "golang.org/x/crypto/bcrypt"
)

type User struct {
    ID           int
    Username     string
    PasswordHash string
    Role         string
}

var ErrNotFound = errors.New("user not found")

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
    u := &User{}
    err := db.QueryRow(
        "SELECT id, username, password_hash, role FROM users WHERE username = ?",
        username,
    ).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    return u, nil
}

// CreateUser hashes the password with bcrypt and inserts the user.
func CreateUser(db *sql.DB, username, password, role string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    _, err = db.Exec(
        "INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
        username, string(hash), role,
    )
    return err
}

func CountUsers(db *sql.DB) (int, error) {
    var n int
    err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
    return n, err
}

// SeedAdmin creates the default admin account if there are no users yet.
func SeedAdmin(db *sql.DB, password string) error {
    n, err := CountUsers(db)
    if err != nil {
        return err
    }
    if n > 0 {
        return nil
    }
    log.Println("No users found: creating default admin account (username: admin)")
    return CreateUser(db, "admin", password, "admin")
}
