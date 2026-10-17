package database

import (
    "database/sql"
    _ "embed"
    "fmt"

    _ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Connect opens the SQLite database file (it is created if it does not exist).
func Connect(path string) (*sql.DB, error) {
    dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)

    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}

// Migrate creates the tables if they do not exist yet.
func Migrate(db *sql.DB) error {
    _, err := db.Exec(schema)
    return err
}
