package database

import (
    "database/sql"
    _ "embed"
)

//go:embed schema_fees.sql
var feesSchema string

// MigrateFees creates the fee, charge and payment tables if they do not exist yet.
func MigrateFees(db *sql.DB) error {
    _, err := db.Exec(feesSchema)
    return err
}
