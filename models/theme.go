package models

import "database/sql"

// GetTheme returns the user's saved theme: "light", "dark", or "" if they have not chosen yet.
func GetTheme(db *sql.DB, userID int) (string, error) {
    var theme string
    err := db.QueryRow("SELECT theme FROM users WHERE id = ?", userID).Scan(&theme)
    return theme, err
}

func SetTheme(db *sql.DB, userID int, theme string) error {
    _, err := db.Exec("UPDATE users SET theme = ? WHERE id = ?", theme, userID)
    return err
}
