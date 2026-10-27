package handlers

import (
    "bytes"
    "errors"
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "io"
    "net/http"
    "strings"
    "unicode"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const (
    maxLogoBytes  = 1 << 20 // 1 MB
    maxLogoPixels = 2000
    defaultAccent = "#2563eb"
)

type Preset struct {
    Name string
    Hex  string
}

var accentPresets = []Preset{
    {"Blue", "#2563eb"},
    {"Navy", "#1e3a8a"},
    {"Teal", "#0f766e"},
    {"Green", "#15803d"},
    {"Maroon", "#9f1239"},
    {"Red", "#b91c1c"},
    {"Orange", "#c2410c"},
    {"Purple", "#7c3aed"},
    {"Gold", "#a16207"},
}

type SettingsForm struct {
    Name       string
    Accent     string
    UseDefault bool
}

type SettingsData struct {
    Form        SettingsForm
    AccentInput string
    HasLogo     bool
    LogoVersion string
    Presets     []Preset
    Notice      string
}

// schoolInfo loads the school's name, logo and colours for the page being rendered.
func (a *App) schoolInfo() SchoolInfo {
    s, err := models.GetSchool(a.DB)
    if err != nil {
        s = models.School{Name: models.DefaultSchoolName}
    }
    return buildSchoolInfo(s)
}

func (a *App) settingsForm() (SettingsForm, error) {
    s, err := models.GetSchool(a.DB)
    if err != nil {
        return SettingsForm{}, err
    }
    return SettingsForm{Name: s.Name, Accent: s.Accent, UseDefault: s.Accent == ""}, nil
}

func (a *App) renderSettings(w http.ResponseWriter, r *http.Request, status int, form SettingsForm, msg, notice string) {
    school, err := models.GetSchool(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    input := form.Accent
    if _, ok := parseHex(input); !ok {
        input = defaultAccent
    }
    a.render(w, r, "settings", status, PageData{
        Title: "School settings",
        Error: msg,
        Data: SettingsData{
            Form:        form,
            AccentInput: input,
            HasLogo:     school.HasLogo,
            LogoVersion: school.LogoVersion,
            Presets:     accentPresets,
            Notice:      notice,
        },
    })
}

func (a *App) SettingsPage(w http.ResponseWriter, r *http.Request) {
    form, err := a.settingsForm()
    if err != nil {
        a.serverError(w, err)
        return
    }
    notice := ""
    switch r.URL.Query().Get("notice") {
    case "saved":
        notice = "Settings saved."
    case "logo":
        notice = "The logo was updated."
    case "logo-removed":
        notice = "The logo was removed."
    }
    a.renderSettings(w, r, http.StatusOK, form, "", notice)
}

func (a *App) SettingsSave(w http.ResponseWriter, r *http.Request) {
    f := SettingsForm{
        Name:       strings.TrimSpace(r.PostFormValue("name")),
        Accent:     strings.ToLower(strings.TrimSpace(r.PostFormValue("accent"))),
        UseDefault: r.PostFormValue("use_default") == "1",
    }

    msg := ""
    switch {
    case f.Name == "" || utf8.RuneCountInString(f.Name) > 60:
        msg = "The school name is required (max 60 characters)"
    case strings.IndexFunc(f.Name, unicode.IsControl) >= 0:
        msg = "The school name contains characters that are not allowed"
    }
    if msg == "" && !f.UseDefault {
        if _, ok := parseHex(f.Accent); !ok {
            msg = "Please choose a valid colour"
        }
    }
    if f.UseDefault {
        f.Accent = ""
    }
    if msg != "" {
        a.renderSettings(w, r, http.StatusBadRequest, f, msg, "")
        return
    }

    if err := models.SaveSchoolSettings(a.DB, f.Name, f.Accent); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/settings?notice=saved", http.StatusSeeOther)
}

// logoError shows the settings page again with a message about the upload.
func (a *App) logoError(w http.ResponseWriter, r *http.Request, msg string) {
    form, err := a.settingsForm()
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderSettings(w, r, http.StatusBadRequest, form, msg, "")
}

func (a *App) LogoUpload(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, maxLogoBytes+(64<<10))
    if err := r.ParseMultipartForm(maxLogoBytes); err != nil {
        a.logoError(w, r, "The upload failed or the file is too large (max 1 MB)")
        return
    }
    if r.MultipartForm != nil {
        defer r.MultipartForm.RemoveAll()
    }

    file, _, err := r.FormFile("logo")
    if err != nil {
        a.logoError(w, r, "Please choose an image file")
        return
    }
    defer file.Close()

    data, err := io.ReadAll(io.LimitReader(file, maxLogoBytes+1))
    if err != nil {
        a.logoError(w, r, "The file could not be read")
        return
    }
    if len(data) == 0 {
        a.logoError(w, r, "The file is empty")
        return
    }
    if len(data) > maxLogoBytes {
        a.logoError(w, r, "The file is too large (max 1 MB)")
        return
    }

    contentType := http.DetectContentType(data)
    switch contentType {
    case "image/png", "image/jpeg", "image/gif":
    default:
        a.logoError(w, r, "Only PNG, JPEG or GIF images are allowed")
        return
    }
    cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
    if err != nil {
        a.logoError(w, r, "That file is not a valid image")
        return
    }
    if cfg.Width > maxLogoPixels || cfg.Height > maxLogoPixels {
        a.logoError(w, r, fmt.Sprintf("The image is too large (max %d x %d pixels)", maxLogoPixels, maxLogoPixels))
        return
    }

    if err := models.SetLogo(a.DB, contentType, data); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/settings?notice=logo", http.StatusSeeOther)
}

func (a *App) LogoRemove(w http.ResponseWriter, r *http.Request) {
    if err := models.DeleteLogo(a.DB); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/settings?notice=logo-removed", http.StatusSeeOther)
}

// LogoServe sends the school logo. It is public, because the login page shows it too.
func (a *App) LogoServe(w http.ResponseWriter, r *http.Request) {
    contentType, data, err := models.GetLogo(a.DB)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    w.Header().Set("Content-Type", contentType)
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
    w.Header().Set("Cache-Control", "public, max-age=86400")
    w.Write(data)
}
