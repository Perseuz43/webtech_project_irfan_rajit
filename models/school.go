package models

import (
    "database/sql"
    "errors"
    "strconv"
    "strings"
    "time"
)

const DefaultSchoolName = "School Management System"

type School struct {
    Name        string
    Accent      string // "#rrggbb", or "" for the default blue
    HasLogo     bool
    LogoVersion string
}

func GetSchool(db *sql.DB) (School, error) {
    s := School{Name: DefaultSchoolName}

    rows, err := db.Query("SELECT key, value FROM settings WHERE key IN ('school_name', 'accent_color')")
    if err != nil {
        return s, err
    }
    for rows.Next() {
        var k, v string
        if err := rows.Scan(&k, &v); err != nil {
            rows.Close()
            return s, err
        }
        switch k {
        case "school_name":
            if strings.TrimSpace(v) != "" {
                s.Name = v
            }
        case "accent_color":
            s.Accent = v
        }
    }
    if err := rows.Err(); err != nil {
        rows.Close()
        return s, err
    }
    rows.Close()

    var version string
    err = db.QueryRow("SELECT version FROM school_logo WHERE id = 1").Scan(&version)
    switch {
    case err == nil:
        s.HasLogo = true
        s.LogoVersion = version
    case errors.Is(err, sql.ErrNoRows):
    default:
        return s, err
    }
    return s, nil
}

func setSetting(tx *sql.Tx, key, value string) error {
    _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
        ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
    return err
}

// SaveSchoolSettings saves the school name and accent colour ("" = default).
func SaveSchoolSettings(db *sql.DB, name, accent string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if err := setSetting(tx, "school_name", name); err != nil {
        return err
    }
    if err := setSetting(tx, "accent_color", accent); err != nil {
        return err
    }
    return tx.Commit()
}

func SetLogo(db *sql.DB, contentType string, data []byte) error {
    _, err := db.Exec(`INSERT INTO school_logo (id, content_type, data, version) VALUES (1, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET content_type = excluded.content_type,
            data = excluded.data, version = excluded.version`,
        contentType, data, strconv.FormatInt(time.Now().UnixNano(), 10))
    return err
}

func GetLogo(db *sql.DB) (string, []byte, error) {
    var contentType string
    var data []byte
    err := db.QueryRow("SELECT content_type, data FROM school_logo WHERE id = 1").Scan(&contentType, &data)
    if err != nil {
        return "", nil, notFoundIf(err)
    }
    return contentType, data, nil
}

func DeleteLogo(db *sql.DB) error {
    _, err := db.Exec("DELETE FROM school_logo WHERE id = 1")
    return err
}
