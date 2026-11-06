package handlers

import (
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const homeworkPastDays = 14 // finished homework stays visible this many days

type HomeworkListData struct {
    Heading  string
    Back     string
    Upcoming []models.Homework
    Past     []models.Homework
    Staff    bool // may post homework
    Today    string
    ClassID  int // the class filter (staff pages)
    Classes  []models.Class
}

type HomeworkFormData struct {
    Edit     bool
    ID       int
    Subjects []models.HomeworkSubject
    SubjectID int
    ClassName string // shown on the edit page
    SubjectName string
    Title    string
    Details  string
    DueDate  string
    Tomorrow string
    MaxTitle int
    MaxDetails int
}

func dateString(t time.Time) string { return t.Format("2006-01-02") }

// canChangeHomework: admins may change any homework, teachers only what they posted.
func canChangeHomework(u *SessionUser, h *models.Homework) bool {
    return u.Role == "admin" || (u.Role == "teacher" && h.PostedBy == u.ID)
}

// splitHomework separates what is still to do from what is already finished.
func splitHomework(list []models.Homework, today string, u *SessionUser) (upcoming, past []models.Homework) {
    for _, h := range list {
        if u != nil {
            h.Editable = canChangeHomework(u, &h)
        }
        if h.DueDate >= today {
            upcoming = append(upcoming, h)
        } else {
            past = append(past, h)
        }
    }
    // the past list shows the most recent first
    for i, j := 0, len(past)-1; i < j; i, j = i+1, j-1 {
        past[i], past[j] = past[j], past[i]
    }
    return
}

func (a *App) renderHomeworkList(w http.ResponseWriter, r *http.Request, data HomeworkListData, classIDs []int) {
    now := time.Now()
    data.Today = dateString(now)
    since := dateString(now.AddDate(0, 0, -homeworkPastDays))
    list, err := models.ListHomework(a.DB, classIDs, since)
    if err != nil {
        a.serverError(w, err)
        return
    }
    var viewer *SessionUser
    if data.Staff {
        viewer = a.currentUser(r)
    }
    data.Upcoming, data.Past = splitHomework(list, data.Today, viewer)
    a.render(w, r, "homework", http.StatusOK, PageData{Title: "Homework", Data: data})
}

// HomeworkList shows homework to the person who opened it: a student sees their class, teachers
// and admins see the classes they work with (and can post and change homework).
func (a *App) HomeworkList(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    switch u.Role {
    case "student":
        s, ok := a.currentStudent(w, r)
        if !ok {
            return
        }
        a.studentHomework(w, r, s, "Homework")
    case "parent":
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
    case "admin", "teacher":
        a.staffHomework(w, r, u)
    default:
        a.NotFound(w, r)
    }
}

func (a *App) studentHomework(w http.ResponseWriter, r *http.Request, s *models.Student, heading string) {
    classIDs := []int{}
    if s.ClassID != 0 {
        classIDs = append(classIDs, s.ClassID)
    }
    a.renderHomeworkList(w, r, HomeworkListData{Heading: heading, Back: "/dashboard"}, classIDs)
}

func (a *App) staffHomework(w http.ResponseWriter, r *http.Request, u *SessionUser) {
    subjects, err := models.HomeworkSubjects(a.DB, u.Role, u.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    // the classes this person can see homework for
    seen := make(map[int]bool)
    var classes []models.Class
    var classIDs []int
    for _, s := range subjects {
        if !seen[s.ClassID] {
            seen[s.ClassID] = true
            classes = append(classes, models.Class{ID: s.ClassID, Name: s.ClassName})
            classIDs = append(classIDs, s.ClassID)
        }
    }
    data := HomeworkListData{Heading: "Homework", Back: "/dashboard", Staff: true, Classes: classes}

    if n, err := strconv.Atoi(r.URL.Query().Get("class_id")); err == nil && seen[n] {
        data.ClassID = n
        classIDs = []int{n}
    }
    if u.Role == "admin" && data.ClassID == 0 {
        classIDs = nil // admins see every class when no filter is chosen
    } else if classIDs == nil {
        classIDs = []int{} // a teacher with no subjects sees nothing
    }
    a.renderHomeworkList(w, r, data, classIDs)
}

// ChildHomework shows a linked child's class homework to their parent.
func (a *App) ChildHomework(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        a.studentHomework(w, r, s, s.Name+": homework")
    }
}

func (a *App) renderHomeworkForm(w http.ResponseWriter, r *http.Request, status int, data HomeworkFormData, msg string) {
    data.Tomorrow = dateString(time.Now().AddDate(0, 0, 1))
    data.MaxTitle, data.MaxDetails = models.MaxHomeworkTitle, models.MaxHomeworkDetails
    if data.DueDate == "" {
        data.DueDate = data.Tomorrow
    }
    title := "Post homework"
    if data.Edit {
        title = "Edit homework"
    }
    a.render(w, r, "homework_form", status, PageData{Title: title, Error: msg, Data: data})
}

func (a *App) HomeworkNewForm(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    subjects, err := models.HomeworkSubjects(a.DB, u.Role, u.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.renderHomeworkForm(w, r, http.StatusOK, HomeworkFormData{Subjects: subjects}, "")
}

// validateHomeworkText checks the parts shared by posting and editing.
func validateHomeworkText(title, details, due string, today time.Time, existingDue string) string {
    switch {
    case title == "":
        return "Please give the homework a title"
    case utf8.RuneCountInString(title) > models.MaxHomeworkTitle:
        return "The title is too long (max " + strconv.Itoa(models.MaxHomeworkTitle) + " characters)"
    case utf8.RuneCountInString(details) > models.MaxHomeworkDetails:
        return "The details are too long (max " + strconv.Itoa(models.MaxHomeworkDetails) + " characters)"
    }
    d, err := time.ParseInLocation("2006-01-02", due, today.Location())
    if err != nil {
        return "Please choose a due date"
    }
    if due != existingDue && dateString(d) < dateString(today) {
        return "The due date cannot be in the past"
    }
    if d.After(today.AddDate(1, 0, 0)) {
        return "The due date is too far away (within one year, please)"
    }
    return ""
}

func (a *App) HomeworkCreate(w http.ResponseWriter, r *http.Request) {
    u := a.currentUser(r)
    subjects, err := models.HomeworkSubjects(a.DB, u.Role, u.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    sid, _ := strconv.Atoi(r.PostFormValue("subject_id"))
    form := HomeworkFormData{
        Subjects: subjects, SubjectID: sid,
        Title:   strings.TrimSpace(r.PostFormValue("title")),
        Details: strings.TrimSpace(r.PostFormValue("details")),
        DueDate: r.PostFormValue("due_date"),
    }
    fail := func(msg string) { a.renderHomeworkForm(w, r, http.StatusBadRequest, form, msg) }

    var chosen *models.HomeworkSubject
    for i := range subjects {
        if subjects[i].SubjectID == sid {
            chosen = &subjects[i]
        }
    }
    if chosen == nil {
        fail("Please choose one of your subjects")
        return
    }
    if msg := validateHomeworkText(form.Title, form.Details, form.DueDate, time.Now(), ""); msg != "" {
        fail(msg)
        return
    }
    err = models.CreateHomework(a.DB, models.HomeworkInput{
        ClassID: chosen.ClassID, SubjectID: chosen.SubjectID,
        Title: form.Title, Details: form.Details, DueDate: form.DueDate, PostedBy: u.ID,
    })
    if err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/homework", http.StatusSeeOther)
}

// loadChangeableHomework finds the homework in the URL and checks the viewer may change it.
func (a *App) loadChangeableHomework(w http.ResponseWriter, r *http.Request) (*models.Homework, bool) {
    id, ok := pathInt(r, "id")
    if !ok {
        a.NotFound(w, r)
        return nil, false
    }
    h, err := models.GetHomework(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        a.NotFound(w, r)
        return nil, false
    }
    if err != nil {
        a.serverError(w, err)
        return nil, false
    }
    if !canChangeHomework(a.currentUser(r), h) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return nil, false
    }
    return h, true
}

func homeworkEditForm(h *models.Homework) HomeworkFormData {
    return HomeworkFormData{Edit: true, ID: h.ID, ClassName: h.ClassName, SubjectName: h.SubjectName,
        Title: h.Title, Details: h.Details, DueDate: h.DueDate}
}

func (a *App) HomeworkEditForm(w http.ResponseWriter, r *http.Request) {
    if h, ok := a.loadChangeableHomework(w, r); ok {
        a.renderHomeworkForm(w, r, http.StatusOK, homeworkEditForm(h), "")
    }
}

func (a *App) HomeworkUpdate(w http.ResponseWriter, r *http.Request) {
    h, ok := a.loadChangeableHomework(w, r)
    if !ok {
        return
    }
    form := homeworkEditForm(h)
    form.Title = strings.TrimSpace(r.PostFormValue("title"))
    form.Details = strings.TrimSpace(r.PostFormValue("details"))
    form.DueDate = r.PostFormValue("due_date")

    if msg := validateHomeworkText(form.Title, form.Details, form.DueDate, time.Now(), h.DueDate); msg != "" {
        a.renderHomeworkForm(w, r, http.StatusBadRequest, form, msg)
        return
    }
    if err := models.UpdateHomework(a.DB, h.ID, form.Title, form.Details, form.DueDate); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/homework", http.StatusSeeOther)
}

func (a *App) HomeworkDelete(w http.ResponseWriter, r *http.Request) {
    h, ok := a.loadChangeableHomework(w, r)
    if !ok {
        return
    }
    if err := models.DeleteHomework(a.DB, h.ID); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/homework", http.StatusSeeOther)
}

// homeworkDueSoon fills the student dashboard with the next few pieces of homework.
func (a *App) homeworkDueSoon(u *SessionUser, data *DashboardData) error {
    if u.Role != "student" || data.Student == nil || data.Student.ClassID == 0 {
        return nil
    }
    today := dateString(time.Now())
    list, err := models.ListHomework(a.DB, []int{data.Student.ClassID}, today)
    if err != nil {
        return err
    }
    if len(list) > 3 {
        list = list[:3]
    }
    data.HomeworkDue = list
    return nil
}
