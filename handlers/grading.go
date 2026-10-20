package handlers

import (
    "errors"
    "fmt"
    "math"
    "net/http"
    "strconv"
    "strings"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const defaultBands = "80 A+ 5.00\n70 A 4.00\n60 A- 3.50\n50 B 3.00\n40 C 2.00\n33 D 1.00\n0 F 0.00"

type ScaleForm struct {
    Name        string
    PassPercent string
    UseGPA      bool
    Bands       string
}

type ScaleFormData struct {
    Editing bool
    ID      int
    Action  string
    Form    ScaleForm
}

type GradingListData struct {
    Scales   []*models.GradingScale
    ActiveID int
}

func readScaleForm(r *http.Request) ScaleForm {
    return ScaleForm{
        Name:        strings.TrimSpace(r.PostFormValue("name")),
        PassPercent: strings.TrimSpace(r.PostFormValue("pass_percent")),
        UseGPA:      r.PostFormValue("use_gpa") == "1",
        Bands:       r.PostFormValue("bands"),
    }
}

func validateScale(f ScaleForm) (models.ScaleInput, string) {
    var in models.ScaleInput
    if f.Name == "" || utf8.RuneCountInString(f.Name) > 60 {
        return in, "Name is required (max 60 characters)"
    }
    pass, err := strconv.ParseFloat(f.PassPercent, 64)
    if err != nil || math.IsNaN(pass) || pass < 0 || pass > 100 {
        return in, "The pass mark must be a number from 0 to 100"
    }
    bands, err := models.ParseBands(f.Bands, f.UseGPA)
    if err != nil {
        return in, err.Error()
    }
    return models.ScaleInput{Name: f.Name, PassPercent: pass, UseGPA: f.UseGPA, Bands: bands}, ""
}

func (a *App) renderGradingList(w http.ResponseWriter, r *http.Request, status int, msg string) {
    scales, err := models.ListScales(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    activeID := 0
    active, err := models.ActiveScale(a.DB)
    if err == nil {
        activeID = active.ID
    } else if !errors.Is(err, models.ErrNotFound) {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "grading", status, PageData{
        Title: "Grading system",
        Error: msg,
        Data:  GradingListData{Scales: scales, ActiveID: activeID},
    })
}

func (a *App) renderScaleForm(w http.ResponseWriter, r *http.Request, status int, data ScaleFormData, msg string) {
    title := "Create grading system"
    data.Action = "/admin/grading"
    if data.Editing {
        title = "Edit grading system"
        data.Action = "/admin/grading/" + strconv.Itoa(data.ID)
    }
    a.render(w, r, "grading_form", status, PageData{Title: title, Error: msg, Data: data})
}

func (a *App) GradingList(w http.ResponseWriter, r *http.Request) {
    a.renderGradingList(w, r, http.StatusOK, "")
}

func (a *App) GradingSetActive(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PostFormValue("scale_id"))
    if err != nil || id <= 0 {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }
    if err := models.SetActiveScale(a.DB, id); err != nil {
        if errors.Is(err, models.ErrNotFound) {
            http.NotFound(w, r)
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/grading", http.StatusSeeOther)
}

func (a *App) GradingNewForm(w http.ResponseWriter, r *http.Request) {
    a.renderScaleForm(w, r, http.StatusOK, ScaleFormData{
        Form: ScaleForm{PassPercent: "33", UseGPA: true, Bands: defaultBands},
    }, "")
}

func (a *App) GradingCreate(w http.ResponseWriter, r *http.Request) {
    f := readScaleForm(r)
    data := ScaleFormData{Form: f}

    in, msg := validateScale(f)
    if msg != "" {
        a.renderScaleForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.SaveScale(a.DB, 0, in); err != nil {
        if models.IsDuplicate(err) {
            a.renderScaleForm(w, r, http.StatusConflict, data, "A grading system with that name already exists")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/grading", http.StatusSeeOther)
}

// customScale loads a scale for editing or deleting. It answers the request itself when it cannot.
func (a *App) customScale(w http.ResponseWriter, r *http.Request) (*models.GradingScale, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return nil, false
    }
    s, err := models.GetScale(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    if s.IsPreset {
        a.renderGradingList(w, r, http.StatusForbidden,
            "The built-in grading systems cannot be changed. Create your own system to customise one.")
        return nil, false
    }
    return s, true
}

func (a *App) GradingEditForm(w http.ResponseWriter, r *http.Request) {
    s, ok := a.customScale(w, r)
    if !ok {
        return
    }
    a.renderScaleForm(w, r, http.StatusOK, ScaleFormData{
        Editing: true,
        ID:      s.ID,
        Form: ScaleForm{
            Name:        s.Name,
            PassPercent: fmt.Sprintf("%g", s.PassPercent),
            UseGPA:      s.UseGPA,
            Bands:       s.BandsText(),
        },
    }, "")
}

func (a *App) GradingUpdate(w http.ResponseWriter, r *http.Request) {
    s, ok := a.customScale(w, r)
    if !ok {
        return
    }
    f := readScaleForm(r)
    data := ScaleFormData{Editing: true, ID: s.ID, Form: f}

    in, msg := validateScale(f)
    if msg != "" {
        a.renderScaleForm(w, r, http.StatusBadRequest, data, msg)
        return
    }
    if err := models.SaveScale(a.DB, s.ID, in); err != nil {
        if models.IsDuplicate(err) {
            a.renderScaleForm(w, r, http.StatusConflict, data, "A grading system with that name already exists")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/grading", http.StatusSeeOther)
}

func (a *App) GradingDelete(w http.ResponseWriter, r *http.Request) {
    s, ok := a.customScale(w, r)
    if !ok {
        return
    }
    active, err := models.ActiveScale(a.DB)
    if err != nil && !errors.Is(err, models.ErrNotFound) {
        a.serverError(w, err)
        return
    }
    if err == nil && active.ID == s.ID {
        a.renderGradingList(w, r, http.StatusConflict, "That system is in use. Choose another system first, then delete it.")
        return
    }
    if err := models.DeleteScale(a.DB, s.ID); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/grading", http.StatusSeeOther)
}
