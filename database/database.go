package database

import (
    "context"
    "database/sql"
    _ "embed"
    "errors"
    "fmt"
    "strings"

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

// Migrate creates the tables if they do not exist yet, then upgrades older databases.
func Migrate(db *sql.DB) error {
    if _, err := db.Exec(schema); err != nil {
        return err
    }
    if err := addColumnIfMissing(db, "users", "theme",
        "TEXT NOT NULL DEFAULT '' CHECK (theme IN ('', 'light', 'dark'))"); err != nil {
        return err
    }
    return allowParentRole(db)
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

// allowParentRole rebuilds an old users table whose role rule does not include 'parent'.
func allowParentRole(db *sql.DB) error {
    var tableSQL string
    err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'").Scan(&tableSQL)
    if err != nil {
        return err
    }
    if strings.Contains(tableSQL, "'parent'") {
        return nil
    }
    return rebuildUsersTable(db)
}

func rebuildUsersTable(db *sql.DB) error {
    ctx := context.Background()
    conn, err := db.Conn(ctx)
    if err != nil {
        return err
    }
    defer conn.Close()

    // Foreign keys must be off while the table is replaced. Otherwise, dropping the
    // old users table would also delete every student and teacher profile.
    if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
        return err
    }
    defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

    tx, err := conn.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    steps := []string{
        `CREATE TABLE users_new (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            username      TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            role          TEXT NOT NULL CHECK (role IN ('admin', 'teacher', 'student', 'parent')),
            created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            theme         TEXT NOT NULL DEFAULT '' CHECK (theme IN ('', 'light', 'dark'))
        )`,
        `INSERT INTO users_new (id, username, password_hash, role, created_at, theme)
            SELECT id, username, password_hash, role, created_at, theme FROM users`,
        `DROP TABLE users`,
        `ALTER TABLE users_new RENAME TO users`,
    }
    for _, s := range steps {
        if _, err := tx.ExecContext(ctx, s); err != nil {
            return fmt.Errorf("rebuilding users table: %w", err)
        }
    }

    rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
    if err != nil {
        return err
    }
    broken := rows.Next()
    rows.Close()
    if broken {
        return errors.New("foreign key check failed after rebuilding the users table")
    }
    return tx.Commit()
}
