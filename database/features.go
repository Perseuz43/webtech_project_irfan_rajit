package database

import (
    "database/sql"
    _ "embed"
)

// Tables for the modules added after the first release. Each module adds one step here.

//go:embed schema_photos.sql
var photosSchema string

//go:embed schema_details.sql
var detailsSchema string

//go:embed schema_remarks.sql
var remarksSchema string

// MigrateFeatures creates the tables of the extra modules if they do not exist yet.
func MigrateFeatures(db *sql.DB) error {
    steps := []string{
        photosSchema,
        detailsSchema,
        remarksSchema,
    }
    for _, s := range steps {
        if _, err := db.Exec(s); err != nil {
            return err
        }
    }
    return nil
}
