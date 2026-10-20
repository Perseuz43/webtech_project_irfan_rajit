package database

import (
    "database/sql"
    _ "embed"
)

//go:embed schema_exams.sql
var examsSchema string

// MigrateExams creates the grading, exam and marks tables if they do not exist yet.
func MigrateExams(db *sql.DB) error {
    _, err := db.Exec(examsSchema)
    return err
}
