package handlers

import (
    "errors"
    "log"
    "net/http"
    "strconv"
    "strings"

    "golang.org/x/crypto/bcrypt"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type PasswordPageData struct {
    Action   string
    Changed  bool
    Username string // used on the admin reset page
    Role     string
}

type UsersPageData struct {
    Users   []models.UserRow
    Message string
}

func validateNewPassword(username, newPw, confirm string) string {
    switch {
    case len(newPw) < 8:
        return "The new password must be at least 8 characters"
    case len(newPw) > 72:
        return "The new password is too long (max 72 characters)"
    case newPw != confirm:
        return "The two new passwords do not match"
    case strings.EqualFold(newPw, username):
        return "The password cannot be the same as the username"
    }
    return ""
}

// ---------- change my own password ----------

func (a *App) renderPasswordForm(w http.ResponseWriter, r *http.Request, status int, msg string, changed bool) {
    a.render(w, r, "password", status, PageData{
        Title: "Change password",
        Error: msg,
        Data:  PasswordPageData{Action: "/account/password", Changed: changed},
    })
}

func (a *App) PasswordForm(w http.ResponseWriter, r *http.Request) {
    a.renderPasswordForm(w, r, http.StatusOK, "", r.URL.Query().Get("changed") == "1")
}

func (a *App) PasswordChange(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    user, err := models.GetUserByID(a.DB, u.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }

    current := r.PostFormValue("current_password")
    newPw := r.PostFormValue("new_password")
    confirm := r.PostFormValue("confirm_password")

    if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)) != nil {
        a.renderPasswordForm(w, r, http.StatusUnauthorized, "Your current password is not correct", false)
        return
    }
    if msg := validateNewPassword(user.Username, newPw, confirm); msg != "" {
        a.renderPasswordForm(w, r, http.StatusBadRequest, msg, false)
        return
    }
    if newPw == current {
        a.renderPasswordForm(w, r, http.StatusBadRequest, "The new password must be different from the current one", false)
        return
    }

    if err := models.SetPassword(a.DB, user.ID, newPw); err != nil {
        a.serverError(w, err)
        return
    }
    // a fresh session ID after a security change
    if err := a.Sessions.RenewToken(r.Context()); err != nil {
        log.Println("session error:", err)
    }
    http.Redirect(w, r, "/account/password?changed=1", http.StatusSeeOther)
}

// ---------- admin: list users and reset a password ----------

func (a *App) UserList(w http.ResponseWriter, r *http.Request) {
    users, err := models.ListUsers(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }

    message := ""
    if id, err := strconv.Atoi(r.URL.Query().Get("reset")); err == nil && id > 0 {
        if target, err := models.GetUserByID(a.DB, id); err == nil {
            message = "The password for " + target.Username + " was reset."
        }
    }
    a.render(w, r, "users", http.StatusOK, PageData{
        Title: "Users and passwords",
        Data:  UsersPageData{Users: users, Message: message},
    })
}

// resetTarget loads the account the admin wants to reset. It answers the request itself when it cannot.
func (a *App) resetTarget(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return nil, false
    }
    // the admin's own password is changed on the normal page, which asks for the current password
    if me := a.currentUser(r); me != nil && me.ID == id {
        http.Redirect(w, r, "/account/password", http.StatusSeeOther)
        return nil, false
    }
    target, err := models.GetUserByID(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    return target, true
}

func (a *App) renderResetForm(w http.ResponseWriter, r *http.Request, status int, target *models.User, msg string) {
    a.render(w, r, "user_password", status, PageData{
        Title: "Reset password",
        Error: msg,
        Data: PasswordPageData{
            Action:   "/admin/users/" + strconv.Itoa(target.ID) + "/password",
            Username: target.Username,
            Role:     target.Role,
        },
    })
}

func (a *App) UserPasswordForm(w http.ResponseWriter, r *http.Request) {
    if target, ok := a.resetTarget(w, r); ok {
        a.renderResetForm(w, r, http.StatusOK, target, "")
    }
}

func (a *App) UserPasswordSet(w http.ResponseWriter, r *http.Request) {
    target, ok := a.resetTarget(w, r)
    if !ok {
        return
    }
    newPw := r.PostFormValue("new_password")
    confirm := r.PostFormValue("confirm_password")
    if msg := validateNewPassword(target.Username, newPw, confirm); msg != "" {
        a.renderResetForm(w, r, http.StatusBadRequest, target, msg)
        return
    }
    if err := models.SetPassword(a.DB, target.ID, newPw); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/users?reset="+strconv.Itoa(target.ID), http.StatusSeeOther)
}
