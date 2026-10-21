package handlers

import (
    "errors"
    "net/http"
    "strconv"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

type ChargeWithPayments struct {
    Charge   models.ChargeRow
    Payments []models.Payment
}

type FeesPageData struct {
    Heading     string
    ReceiptBase string
    Items       []ChargeWithPayments
    Total       models.Money
    Paid        models.Money
    Due         models.Money
}

// feesPage lists one student's bills and payments (used for a student and for a parent's child).
func (a *App) feesPage(w http.ResponseWriter, r *http.Request, s *models.Student, heading, receiptBase string) {
    charges, err := models.ListStudentCharges(a.DB, s.ID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    items := make([]ChargeWithPayments, 0, len(charges))
    for _, c := range charges {
        pays, err := models.ListPayments(a.DB, c.ID)
        if err != nil {
            a.serverError(w, err)
            return
        }
        items = append(items, ChargeWithPayments{Charge: c, Payments: pays})
    }
    total, paid, due := models.SummarizeCharges(charges)

    a.render(w, r, "fees_list", http.StatusOK, PageData{
        Title: "Fees",
        Data: FeesPageData{
            Heading: heading, ReceiptBase: receiptBase, Items: items,
            Total: total, Paid: paid, Due: due,
        },
    })
}

// studentReceipt shows a receipt only if the payment belongs to this student.
func (a *App) studentReceipt(w http.ResponseWriter, r *http.Request, s *models.Student, back string) {
    pid, ok := pathInt(r, "pid")
    if !ok {
        http.NotFound(w, r)
        return
    }
    pd, err := models.GetPayment(a.DB, pid)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    if pd.StudentID != s.ID {
        http.NotFound(w, r)
        return
    }
    a.renderReceipt(w, r, pd, back)
}

func (a *App) MyFees(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.currentStudent(w, r); ok {
        a.feesPage(w, r, s, "My fees", "/my-fees/receipts")
    }
}

func (a *App) MyReceipt(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.currentStudent(w, r); ok {
        a.studentReceipt(w, r, s, "/my-fees")
    }
}

func (a *App) ChildFees(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        base := "/children/" + strconv.Itoa(s.ID) + "/fees"
        a.feesPage(w, r, s, s.Name+": fees", base+"/receipts")
    }
}

func (a *App) ChildReceipt(w http.ResponseWriter, r *http.Request) {
    if s, ok := a.linkedChild(w, r); ok {
        a.studentReceipt(w, r, s, "/children/"+strconv.Itoa(s.ID)+"/fees")
    }
}
