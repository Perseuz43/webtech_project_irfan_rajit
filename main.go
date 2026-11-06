package main

import (
    "log"
    "net/http"
    "time"

    "github.com/alexedwards/scs/v2"

    "github.com/Perseuz43/webtech_project_irfan_rajit/config"
    "github.com/Perseuz43/webtech_project_irfan_rajit/database"
    "github.com/Perseuz43/webtech_project_irfan_rajit/handlers"
    "github.com/Perseuz43/webtech_project_irfan_rajit/middleware"
    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

// noCache tells browsers to check with the server before reusing a file.
func noCache(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "no-cache")
        next.ServeHTTP(w, r)
    })
}

func main() {
    cfg := config.Load()

    db, err := database.Connect(cfg.DBPath)
    if err != nil {
        log.Fatal("database connection failed: ", err)
    }
    defer db.Close()

    if err := database.Migrate(db); err != nil {
        log.Fatal("database migration failed: ", err)
    }
    if err := database.MigrateExams(db); err != nil {
        log.Fatal("exam tables migration failed: ", err)
    }
    if err := database.MigrateNotices(db); err != nil {
        log.Fatal("notices table migration failed: ", err)
    }
    if err := database.MigrateFees(db); err != nil {
        log.Fatal("fee tables migration failed: ", err)
    }
    if err := database.MigrateSchool(db); err != nil {
        log.Fatal("school table migration failed: ", err)
    }
    log.Println("Database connected and migrated:", cfg.DBPath)

    if err := database.MigrateFeatures(db); err != nil {
        log.Fatal("module tables migration failed: ", err)
    }
    if err := models.SeedAdmin(db, cfg.AdminPassword); err != nil {
        log.Fatal("creating default admin failed: ", err)
    }
    if err := models.SeedGrading(db); err != nil {
        log.Fatal("creating grading systems failed: ", err)
    }

    sm := scs.New()
    sm.Lifetime = 12 * time.Hour

    app, err := handlers.New(db, sm)
    if err != nil {
        log.Fatal("loading templates failed: ", err)
    }

    requireLogin := middleware.RequireLogin(sm)
    requireAdmin := middleware.RequireRole(sm, "admin")
    requireStaff := middleware.RequireRole(sm, "admin", "teacher")
    requireStudent := middleware.RequireRole(sm, "student")
    requireParent := middleware.RequireRole(sm, "parent")

    login := func(h http.HandlerFunc) http.Handler { return requireLogin(h) }
    admin := func(h http.HandlerFunc) http.Handler { return requireAdmin(h) }
    staff := func(h http.HandlerFunc) http.Handler { return requireStaff(h) }
    student := func(h http.HandlerFunc) http.Handler { return requireStudent(h) }
    parent := func(h http.HandlerFunc) http.Handler { return requireParent(h) }

    mux := http.NewServeMux()

    // serve CSS/JS files
    // (no-cache means the browser re-checks the file each time, so edits show up right away)
    mux.Handle("GET /static/", noCache(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))

    // public pages
    mux.HandleFunc("GET /{$}", app.Home)
    mux.HandleFunc("GET /login", app.LoginForm)
    mux.HandleFunc("POST /login", app.Login)
    mux.HandleFunc("POST /logout", app.Logout)
    mux.HandleFunc("POST /theme", app.SetTheme)
    mux.HandleFunc("GET /logo", app.LogoServe)

    // logged-in pages (every role)
    mux.Handle("GET /dashboard", login(app.Dashboard))
    mux.Handle("GET /notices", login(app.NoticeList))
    mux.Handle("GET /account/password", login(app.PasswordForm))
    mux.Handle("POST /account/password", login(app.PasswordChange))

    // admin pages
    mux.Handle("GET /admin", admin(app.AdminHome))

    mux.Handle("GET /admin/settings", admin(app.SettingsPage))
    mux.Handle("POST /admin/settings", admin(app.SettingsSave))
    mux.Handle("POST /admin/settings/logo", admin(app.LogoUpload))
    mux.Handle("POST /admin/settings/logo/delete", admin(app.LogoRemove))

    mux.Handle("GET /admin/users", admin(app.UserList))
    mux.Handle("GET /admin/users/{id}/password", admin(app.UserPasswordForm))
    mux.Handle("POST /admin/users/{id}/password", admin(app.UserPasswordSet))

    mux.Handle("GET /admin/classes", admin(app.ClassList))
    mux.Handle("POST /admin/classes", admin(app.ClassCreate))
    mux.Handle("GET /admin/classes/{id}", admin(app.ClassDetail))
    mux.Handle("POST /admin/classes/{id}/rename", admin(app.ClassRename))
    mux.Handle("POST /admin/classes/{id}/delete", admin(app.ClassDelete))
    mux.Handle("POST /admin/classes/{id}/sections", admin(app.SectionCreate))
    mux.Handle("POST /admin/classes/{id}/sections/{sid}/delete", admin(app.SectionDelete))
    mux.Handle("POST /admin/classes/{id}/subjects", admin(app.SubjectCreate))
    mux.Handle("POST /admin/classes/{id}/subjects/{sid}/delete", admin(app.SubjectDelete))

    mux.Handle("GET /admin/students", admin(app.StudentList))
    mux.Handle("GET /admin/students/new", admin(app.StudentNewForm))
    mux.Handle("POST /admin/students", admin(app.StudentCreate))
    mux.Handle("GET /admin/students/{id}/edit", admin(app.StudentEditForm))
    mux.Handle("POST /admin/students/{id}", admin(app.StudentUpdate))
    mux.Handle("POST /admin/students/{id}/delete", admin(app.StudentDelete))
    mux.Handle("GET /admin/students/export.csv", admin(app.StudentExport))
    mux.Handle("GET /admin/students/{id}/details", admin(app.StudentDetails))
    mux.Handle("GET /admin/students/{id}/id-card", admin(app.IDCardOne))
    mux.Handle("POST /admin/students/{id}/dob", admin(app.StudentDOBSave))
    mux.Handle("GET /admin/id-cards", admin(app.IDCardsPage))
    mux.Handle("POST /admin/students/{id}/photo", admin(app.StudentPhotoUpload))
    mux.Handle("POST /admin/students/{id}/photo/delete", admin(app.StudentPhotoRemove))
    mux.Handle("GET /admin/students/import", admin(app.StudentImportForm))
    mux.Handle("POST /admin/students/import", admin(app.StudentImportRun))
    mux.Handle("GET /admin/students/import/sample.csv", admin(app.StudentImportSample))

    mux.Handle("GET /admin/teachers", admin(app.TeacherList))
    mux.Handle("GET /admin/teachers/new", admin(app.TeacherNewForm))
    mux.Handle("POST /admin/teachers", admin(app.TeacherCreate))
    mux.Handle("GET /admin/teachers/{id}/edit", admin(app.TeacherEditForm))
    mux.Handle("POST /admin/teachers/{id}", admin(app.TeacherUpdate))
    mux.Handle("POST /admin/teachers/{id}/subjects", admin(app.TeacherSubjects))
    mux.Handle("POST /admin/teachers/{id}/delete", admin(app.TeacherDelete))

    mux.Handle("GET /admin/parents", admin(app.ParentList))
    mux.Handle("GET /admin/parents/new", admin(app.ParentNewForm))
    mux.Handle("POST /admin/parents", admin(app.ParentCreate))
    mux.Handle("GET /admin/parents/{id}/edit", admin(app.ParentEditForm))
    mux.Handle("POST /admin/parents/{id}", admin(app.ParentUpdate))
    mux.Handle("POST /admin/parents/{id}/children", admin(app.ParentChildren))
    mux.Handle("POST /admin/parents/{id}/delete", admin(app.ParentDelete))

    // grading systems and exams (admin)
    mux.Handle("GET /admin/grading", admin(app.GradingList))
    mux.Handle("POST /admin/grading/active", admin(app.GradingSetActive))
    mux.Handle("GET /admin/grading/new", admin(app.GradingNewForm))
    mux.Handle("POST /admin/grading", admin(app.GradingCreate))
    mux.Handle("GET /admin/grading/{id}/edit", admin(app.GradingEditForm))
    mux.Handle("POST /admin/grading/{id}", admin(app.GradingUpdate))
    mux.Handle("POST /admin/grading/{id}/delete", admin(app.GradingDelete))

    mux.Handle("GET /admin/exams", admin(app.ExamList))
    mux.Handle("POST /admin/exams", admin(app.ExamCreate))
    mux.Handle("GET /admin/exams/{id}", admin(app.ExamDetail))
    mux.Handle("POST /admin/exams/{id}/subjects", admin(app.ExamSaveSubjects))
    mux.Handle("POST /admin/exams/{id}/publish", admin(app.ExamPublish))
    mux.Handle("POST /admin/exams/{id}/delete", admin(app.ExamDelete))
    mux.Handle("GET /admin/exams/{id}/students/{sid}", admin(app.ExamStudentCard))

    // fees (admin)
    mux.Handle("GET /admin/fees", admin(app.FeeList))
    mux.Handle("POST /admin/fees", admin(app.FeeCreate))
    mux.Handle("GET /admin/fees/{id}", admin(app.FeeDetail))
    mux.Handle("POST /admin/fees/{id}/sync", admin(app.FeeBillMissing))
    mux.Handle("POST /admin/fees/{id}/delete", admin(app.FeeDelete))
    mux.Handle("GET /admin/charges/{id}", admin(app.ChargeDetail))
    mux.Handle("POST /admin/charges/{id}/payments", admin(app.PaymentCreate))
    mux.Handle("POST /admin/charges/{id}/payments/{pid}/delete", admin(app.PaymentDelete))
    mux.Handle("GET /admin/payments/{pid}/receipt", admin(app.ReceiptAdmin))
    mux.Handle("GET /admin/dues", admin(app.DuesReport))
    mux.Handle("GET /admin/dues/export.csv", admin(app.DuesExport))

    // notices (admin writes, everyone reads their own on /notices)
    mux.Handle("GET /admin/notices/new", admin(app.NoticeNewForm))
    mux.Handle("POST /admin/notices", admin(app.NoticeCreate))
    mux.Handle("GET /admin/notices/{id}/edit", admin(app.NoticeEditForm))
    mux.Handle("POST /admin/notices/{id}", admin(app.NoticeUpdate))
    mux.Handle("POST /admin/notices/{id}/delete", admin(app.NoticeDelete))

    // student directory: admins see everyone, teachers only their own classes
    mux.Handle("GET /students", staff(app.StudentDirectory))
    mux.Handle("GET /birthdays", staff(app.BirthdaysPage))
    mux.Handle("GET /students/{id}/photo", login(app.StudentPhotoServe))

    // attendance: admins and teachers mark and report, students view their own
    mux.Handle("GET /attendance", staff(app.AttendancePage))
    mux.Handle("POST /attendance", staff(app.AttendanceSave))
    mux.Handle("GET /attendance/report", staff(app.AttendanceReport))
    mux.Handle("GET /my-attendance", student(app.MyAttendance))

    // marks entry: admins and teachers (teachers only for their own subjects)
    mux.Handle("GET /remarks", staff(app.RemarksPage))
    mux.Handle("POST /remarks", staff(app.RemarksSave))
    mux.Handle("GET /marks", staff(app.MarksPage))
    mux.Handle("POST /marks", staff(app.MarksSave))

    // students: their own results and fees
    mux.Handle("GET /my-results", student(app.MyResults))
    mux.Handle("GET /my-results/{eid}", student(app.MyReportCard))
    mux.Handle("GET /my-fees", student(app.MyFees))
    mux.Handle("GET /my-fees/receipts/{pid}", student(app.MyReceipt))

    // parents: only their own linked children
    mux.Handle("GET /children/{id}/attendance", parent(app.ChildAttendance))
    mux.Handle("GET /children/{id}/results", parent(app.ChildResults))
    mux.Handle("GET /children/{id}/results/{eid}", parent(app.ChildReportCard))
    mux.Handle("GET /children/{id}/fees", parent(app.ChildFees))
    mux.Handle("GET /children/{id}/fees/receipts/{pid}", parent(app.ChildReceipt))

    // browsers ask for /favicon.ico on their own; point them at our icon
    mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/static/favicon.svg", http.StatusFound)
    })

    // anything that matches no other route gets a friendly 404 page
    mux.HandleFunc("/", app.NotFound)

    // block cross-site form submissions (CSRF protection)
    cop := http.NewCrossOriginProtection()

    log.Println("Server running on http://localhost:" + cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, sm.LoadAndSave(cop.Handler(mux))))
}
