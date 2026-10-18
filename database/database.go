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

// Migrate creates the tables if they do not exist yet, then adds newer columns to old databases.
func Migrate(db *sql.DB) error {
    if _, err := db.Exec(schema); err != nil {
        return err
    }
    return addColumnIfMissing(db, "users", "theme",
        "TEXT NOT NULL DEFAULT '' CHECK (theme IN ('', 'light', 'dark'))")
}

func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
    var n int
    query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?", table)
    if err := db.QueryRow(query, column).Scan(&n); err != nil {
        return err
    }
    if n > 0 {
        return nil
    }
    _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
    return err
}
