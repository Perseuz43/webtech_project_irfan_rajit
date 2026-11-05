package handlers

import (
    "encoding/csv"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

// csvSafe stops spreadsheets from running text as a formula. A cell that starts with
// = or @ (or a + / - that is not just a number or phone number) gets a ' in front.
func csvSafe(s string) string {
    if s == "" {
        return s
    }
    switch s[0] {
    case '=', '@', '\t', '\r':
        return "'" + s
    case '+', '-':
        if strings.Trim(s[1:], "0123456789 .-()") != "" {
            return "'" + s
        }
    }
    return s
}

// sendCSV writes a download. The byte-order mark at the start lets Excel read UTF-8 names.
func sendCSV(w http.ResponseWriter, baseName string, header []string, rows [][]string) {
    filename := baseName + "-" + time.Now().Format("2006-01-02") + ".csv"
    w.Header().Set("Content-Type", "text/csv; charset=utf-8")
    w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
    w.Write([]byte(utf8ByteOrderMk))

    cw := csv.NewWriter(w)
    cw.Write(header)
    for _, row := range rows {
        safe := make([]string, len(row))
        for i, cell := range row {
            safe[i] = csvSafe(cell)
        }
        cw.Write(safe)
    }
    cw.Flush()
}

// StudentExport downloads every student. The columns match the import file (without
// passwords), so an exported list can be edited and imported again.
func (a *App) StudentExport(w http.ResponseWriter, r *http.Request) {
    students, err := models.ListStudents(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    rows := make([][]string, 0, len(students))
    for _, s := range students {
        rows = append(rows, []string{
            s.Name, strconv.Itoa(s.Roll), s.ClassName, s.SectionName,
            s.GuardianName, s.Phone, s.Username,
        })
    }
    sendCSV(w, "students", []string{"name", "roll", "class", "section", "guardian", "phone", "username"}, rows)
}

// DuesExport downloads the dues report, for all classes or the one chosen with ?class_id=.
func (a *App) DuesExport(w http.ResponseWriter, r *http.Request) {
    classID := 0
    if n, err := strconv.Atoi(r.URL.Query().Get("class_id")); err == nil && n > 0 {
        classID = n
    }
    dues, err := models.ListDues(a.DB, classID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    rows := make([][]string, 0, len(dues))
    for _, c := range dues {
        overdue := "no"
        if c.Overdue {
            overdue = "yes"
        }
        rows = append(rows, []string{
            c.StudentName, strconv.Itoa(c.Roll), c.ClassName, c.FeeName, c.DueDate,
            strconv.FormatInt(int64(c.Amount), 10), strconv.FormatInt(int64(c.Paid), 10),
            strconv.FormatInt(int64(c.Due), 10), c.Status, overdue,
        })
    }
    sendCSV(w, "dues", []string{"student", "roll", "class", "fee", "due date", "amount", "paid", "balance", "status", "overdue"}, rows)
}
