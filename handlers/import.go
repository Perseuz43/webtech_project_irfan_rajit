package handlers

import (
    "bytes"
    "encoding/csv"
    "errors"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const (
    maxImportBytes  = 1 << 20 // 1 MB
    maxImportRows   = 1000
    maxImportShown  = 50 // how many problems to list on the page
    utf8ByteOrderMk = "\xef\xbb\xbf"
)

// ImportRowError describes one problem found in the uploaded file.
type ImportRowError struct {
    Line    int // line in the file (0 = the file as a whole)
    Name    string
    Message string
}

type ImportPageData struct {
    Errors     []ImportRowError
    MoreErrors int // problems that were found but not listed
    Imported   int // > 0 after a successful import
    Classes    []models.Class
    Sections   []models.Section
}

func (a *App) renderImport(w http.ResponseWriter, r *http.Request, status int, data ImportPageData, msg string) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    sections, err := models.ListAllSections(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    data.Classes = classes
    data.Sections = sections
    a.render(w, r, "student_import", status, PageData{Title: "Import students", Error: msg, Data: data})
}

func (a *App) StudentImportForm(w http.ResponseWriter, r *http.Request) {
    a.renderImport(w, r, http.StatusOK, ImportPageData{}, "")
}

// StudentImportSample downloads a small example file that already has the right columns.
func (a *App) StudentImportSample(w http.ResponseWriter, r *http.Request) {
    className, section := "Class 1", "A"
    if classes, err := models.ListClasses(a.DB); err == nil && len(classes) > 0 {
        className = classes[0].Name
        section = ""
        if sections, err := models.ListAllSections(a.DB); err == nil {
            for _, s := range sections {
                if s.ClassID == classes[0].ID {
                    section = s.Name
                    break
                }
            }
        }
    }

    var b strings.Builder
    b.WriteString(utf8ByteOrderMk) // lets Excel read the file as UTF-8
    b.WriteString("name,roll,class,section,guardian,phone,username,password\r\n")
    sample := []struct{ name, guardian, phone, user string }{
        {"Rahim Uddin", "Karim Uddin", "01711111111", "rahim.uddin"},
        {"Nusrat Jahan", "Abdul Jahan", "01822222222", "nusrat.jahan"},
        {"Tanvir Ahmed", "Salma Ahmed", "01933333333", "tanvir.ahmed"},
    }
    for i, s := range sample {
        fmt.Fprintf(&b, "%s,%d,%s,%s,%s,%s,%s,%s\r\n",
            s.name, i+1, csvField(className), csvField(section), s.guardian, s.phone, s.user, "change-me-"+strconv.Itoa(i+1))
    }

    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
    w.Header().Set("Content-Disposition", `attachment; filename="students-sample.csv"`)
    io.WriteString(w, b.String())
}

// csvField quotes a value when it contains a comma, quote or line break.
func csvField(s string) string {
    if strings.ContainsAny(s, ",\"\r\n") {
        return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
    }
    return s
}

func (a *App) StudentImportRun(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+(64<<10))
    if err := r.ParseMultipartForm(maxImportBytes); err != nil {
        a.renderImport(w, r, http.StatusBadRequest, ImportPageData{}, "The upload failed or the file is too large (max 1 MB)")
        return
    }
    if r.MultipartForm != nil {
        defer r.MultipartForm.RemoveAll()
    }

    file, _, err := r.FormFile("csv")
    if err != nil {
        a.renderImport(w, r, http.StatusBadRequest, ImportPageData{}, "Please choose a CSV file")
        return
    }
    defer file.Close()

    data, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
    if err != nil || len(data) == 0 {
        a.renderImport(w, r, http.StatusBadRequest, ImportPageData{}, "The file is empty or could not be read")
        return
    }
    if len(data) > maxImportBytes {
        a.renderImport(w, r, http.StatusBadRequest, ImportPageData{}, "The file is too large (max 1 MB)")
        return
    }

    students, problems, err := a.parseStudentCSV(data)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if len(problems) > 0 {
        page := ImportPageData{Errors: problems}
        if len(problems) > maxImportShown {
            page.Errors = problems[:maxImportShown]
            page.MoreErrors = len(problems) - maxImportShown
        }
        a.renderImport(w, r, http.StatusBadRequest, page,
            "Nothing was imported. Please fix the problems below and upload the file again.")
        return
    }

    if err := models.ImportStudents(a.DB, students); err != nil {
        if models.IsDuplicate(err) {
            a.renderImport(w, r, http.StatusConflict, ImportPageData{},
                "A username or roll number was taken while importing. Nothing was imported, so please try again.")
            return
        }
        a.serverError(w, err)
        return
    }
    a.renderImport(w, r, http.StatusOK, ImportPageData{Imported: len(students)}, "")
}

// csvColumns maps each column we understand to the header names we accept for it.
var csvColumns = map[string][]string{
    "name":     {"name", "student name", "student"},
    "roll":     {"roll", "roll number", "roll no"},
    "class":    {"class", "class name"},
    "section":  {"section"},
    "guardian": {"guardian", "guardian name", "guardian_name"},
    "phone":    {"phone", "phone number"},
    "username": {"username", "user name", "login"},
    "password": {"password"},
}

var requiredCSVColumns = []string{"name", "roll", "class", "username", "password"}

// parseStudentCSV reads the whole file and checks every row. The returned problems list is
// empty only when every row is good. The error is only for unexpected database failures.
func (a *App) parseStudentCSV(data []byte) ([]models.StudentInput, []ImportRowError, error) {
    data = bytes.TrimPrefix(data, []byte(utf8ByteOrderMk))

    // Excel in some countries saves with semicolons instead of commas.
    firstLine := data
    if i := bytes.IndexByte(data, '\n'); i >= 0 {
        firstLine = data[:i]
    }
    reader := csv.NewReader(bytes.NewReader(data))
    reader.Comma = ','
    if bytes.Count(firstLine, []byte(";")) > bytes.Count(firstLine, []byte(",")) {
        reader.Comma = ';'
    }
    reader.FieldsPerRecord = -1
    reader.TrimLeadingSpace = true
    reader.LazyQuotes = true

    header, err := reader.Read()
    if err != nil {
        return nil, []ImportRowError{{Message: "The file has no header row"}}, nil
    }
    index := make(map[string]int)
    for i, h := range header {
        h = strings.ToLower(strings.TrimSpace(h))
        for col, names := range csvColumns {
            for _, n := range names {
                if h == n {
                    if _, seen := index[col]; !seen {
                        index[col] = i
                    }
                }
            }
        }
    }
    var missing []string
    for _, col := range requiredCSVColumns {
        if _, ok := index[col]; !ok {
            missing = append(missing, col)
        }
    }
    if len(missing) > 0 {
        return nil, []ImportRowError{{Message: "The first row must name the columns. Missing: " + strings.Join(missing, ", ")}}, nil
    }

    classIDs, err := models.ClassNamesToIDs(a.DB)
    if err != nil {
        return nil, nil, err
    }
    allSections, err := models.ListAllSections(a.DB)
    if err != nil {
        return nil, nil, err
    }
    sectionIDs := make(map[int]map[string]int) // class ID -> lower-case section name -> ID
    for _, s := range allSections {
        if sectionIDs[s.ClassID] == nil {
            sectionIDs[s.ClassID] = make(map[string]int)
        }
        sectionIDs[s.ClassID][strings.ToLower(s.Name)] = s.ID
    }
    takenUsers, err := models.ExistingUsernames(a.DB)
    if err != nil {
        return nil, nil, err
    }
    takenRolls, err := models.ExistingRolls(a.DB)
    if err != nil {
        return nil, nil, err
    }

    cell := func(rec []string, col string) string {
        i, ok := index[col]
        if !ok || i >= len(rec) {
            return ""
        }
        return strings.TrimSpace(rec[i])
    }

    var students []models.StudentInput
    var problems []ImportRowError

    for {
        rec, err := reader.Read()
        if errors.Is(err, io.EOF) {
            break
        }
        line, _ := reader.FieldPos(0)
        if err != nil {
            problems = append(problems, ImportRowError{Line: line, Message: "This row could not be read: " + err.Error()})
            continue
        }
        if allBlank(rec) {
            continue
        }
        if len(students)+len(problems) >= maxImportRows {
            problems = append(problems, ImportRowError{Message: fmt.Sprintf("Too many rows (max %d per file). Split the file and import it in parts.", maxImportRows)})
            break
        }

        name := cell(rec, "name")
        fail := func(msg string) {
            problems = append(problems, ImportRowError{Line: line, Name: name, Message: msg})
        }

        className := cell(rec, "class")
        classID, ok := classIDs[strings.ToLower(className)]
        if !ok {
            fail(fmt.Sprintf("Class %q does not exist. Create it under Classes first.", className))
            continue
        }
        sectionID := 0
        if sec := cell(rec, "section"); sec != "" {
            sectionID, ok = sectionIDs[classID][strings.ToLower(sec)]
            if !ok {
                fail(fmt.Sprintf("Section %q does not exist in %s.", sec, className))
                continue
            }
        }

        in, msg := a.validateStudent(StudentForm{
            Name:         name,
            Roll:         cell(rec, "roll"),
            ClassID:      strconv.Itoa(classID),
            SectionID:    idString(sectionID),
            GuardianName: cell(rec, "guardian"),
            Phone:        cell(rec, "phone"),
            Username:     cell(rec, "username"),
            Password:     rawCell(rec, index, "password"), // passwords are used exactly as typed
        }, false)
        if msg != "" {
            fail(msg)
            continue
        }

        if takenUsers[in.Username] {
            fail(fmt.Sprintf("The username %q is already taken.", in.Username))
            continue
        }
        key := models.RollKey{ClassID: in.ClassID, Roll: in.Roll}
        if takenRolls[key] {
            fail(fmt.Sprintf("Roll %d is already used in %s.", in.Roll, className))
            continue
        }
        takenUsers[in.Username] = true // so a repeat later in the same file is caught
        takenRolls[key] = true
        students = append(students, in)
    }

    if len(students) == 0 && len(problems) == 0 {
        problems = append(problems, ImportRowError{Message: "The file has a header row but no students."})
    }
    return students, problems, nil
}

func rawCell(rec []string, index map[string]int, col string) string {
    i, ok := index[col]
    if !ok || i >= len(rec) {
        return ""
    }
    return rec[i]
}

func allBlank(rec []string) bool {
    for _, c := range rec {
        if strings.TrimSpace(c) != "" {
            return false
        }
    }
    return true
}
