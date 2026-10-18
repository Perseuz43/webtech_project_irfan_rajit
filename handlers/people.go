package handlers

import (
    "regexp"
    "strconv"
)

var (
    usernameRe = regexp.MustCompile(`^[A-Za-z0-9._-]{3,30}$`)
    phoneRe    = regexp.MustCompile(`^[0-9+ -]{0,20}$`)
)

func validPhone(p string) bool {
    return phoneRe.MatchString(p)
}

// validateCredentials checks the username and starting password. It returns an error message, or "".
func validateCredentials(username, password string) string {
    if !usernameRe.MatchString(username) {
        return "Username must be 3-30 characters: letters, numbers, dot, dash or underscore"
    }
    if len(password) < 6 {
        return "Password must be at least 6 characters"
    }
    if len(password) > 72 {
        return "Password is too long (max 72 characters)"
    }
    return ""
}

// idString turns an ID into text for form fields; 0 means "none".
func idString(n int) string {
    if n == 0 {
        return ""
    }
    return strconv.Itoa(n)
}
