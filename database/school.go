package database

import (
    "database/sql"
    _ "embed"
)

//go:embed schema_school.sql
var schoolSchema string

// MigrateSchool creates the table that holds the school logo.
func MigrateSchool(db *sql.DB) error {
    _, err := db.Exec(schoolSchema)
    return err
}
