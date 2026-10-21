package database

import (
    "database/sql"
    _ "embed"
)

//go:embed schema_notices.sql
var noticesSchema string

// MigrateNotices creates the notices table if it does not exist yet.
func MigrateNotices(db *sql.DB) error {
    _, err := db.Exec(noticesSchema)
    return err
}
