package handlers

import (
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

const maxMoney = 10000000

func parseMoney(s string) (models.Money, bool) {
    n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
    if err != nil || n <= 0 || n > maxMoney {
        return 0, false
    }
    return models.Money(n), true
}

type FeeListData struct {
    Fees    []models.FeeRow
    Classes []models.Class
    Today   string
}

type FeeDetailData struct {
    Fee     *models.FeeRow
    Charges []models.ChargeRow
    Added   string
}

type PaymentForm struct {
    Amount string
    Date   string
    Method string
    Note   string
}

type ChargeDetailData struct {
    Charge   *models.ChargeRow
    Payments []models.Payment
    Form     PaymentForm
    Today    string
}

type DuesData struct {
    Rows     []models.ChargeRow
    Classes  []models.Class
    ClassID  int
    TotalDue models.Money
}

type ReceiptPage struct {
    R    *models.PaymentDetail
    Back string
}

// ---------- fees ----------

func (a *App) renderFeeList(w http.ResponseWriter, r *http.Request, status int, msg string) {
    fees, err := models.ListFees(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    a.render(w, r, "fees", status, PageData{
        Title: "Fees",
        Error: msg,
        Data:  FeeListData{Fees: fees, Classes: classes, Today: today()},
    })
}

func (a *App) FeeList(w http.ResponseWriter, r *http.Request) {
    a.renderFeeList(w, r, http.StatusOK, "")
}

func (a *App) FeeCreate(w http.ResponseWriter, r *http.Request) {
    name, msg := cleanName(r)
    amount, amountOK := parseMoney(r.PostFormValue("amount"))
    due := strings.TrimSpace(r.PostFormValue("due_date"))
    classStr := r.PostFormValue("class_id")
    classID := 0

    if msg == "" && !amountOK {
        msg = "Amount must be a whole number above 0"
    }
    if msg == "" {
        if _, err := time.Parse(dateLayout, due); err != nil {
            msg = "Please enter a valid due date"
        }
    }
    if msg == "" && classStr != "all" {
        id, err := strconv.Atoi(classStr)
        if err != nil || id <= 0 {
            msg = "Please choose who this fee is for"
        } else if _, err := models.GetClass(a.DB, id); err != nil {
            msg = "Please choose a valid class"
        } else {
            classID = id
        }
    }
    if msg != "" {
        a.renderFeeList(w, r, http.StatusBadRequest, msg)
        return
    }

    id, billed, err := models.CreateFee(a.DB, models.FeeInput{
        Name: name, Amount: amount, DueDate: due, ClassID: classID,
    })
    if err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, fmt.Sprintf("/admin/fees/%d?added=%d", id, billed), http.StatusSeeOther)
}

func (a *App) FeeDetail(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    fee, err := models.GetFee(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    charges, err := models.ListFeeCharges(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }

    added := ""
    if n, err := strconv.Atoi(r.URL.Query().Get("added")); err == nil && n >= 0 {
        added = fmt.Sprintf("Billed %d new student(s).", n)
    }
    a.render(w, r, "fee_detail", http.StatusOK, PageData{
        Title: fee.Name,
        Data:  FeeDetailData{Fee: fee, Charges: charges, Added: added},
    })
}

func (a *App) FeeBillMissing(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if _, err := models.GetFee(a.DB, id); err != nil {
        if errors.Is(err, models.ErrNotFound) {
            http.NotFound(w, r)
            return
        }
        a.serverError(w, err)
        return
    }
    n, err := models.BillMissingStudents(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, fmt.Sprintf("/admin/fees/%d?added=%d", id, n), http.StatusSeeOther)
}

func (a *App) FeeDelete(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    if err := models.DeleteFee(a.DB, id); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/fees", http.StatusSeeOther)
}

// ---------- one student's bill and its payments ----------

func (a *App) renderCharge(w http.ResponseWriter, r *http.Request, id, status int, form PaymentForm, msg string) {
    c, err := models.GetCharge(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }
    pays, err := models.ListPayments(a.DB, id)
    if err != nil {
        a.serverError(w, err)
        return
    }
    if form.Date == "" {
        form.Date = today()
    }
    if form.Method == "" {
        form.Method = "cash"
    }
    a.render(w, r, "charge_detail", status, PageData{
        Title: c.FeeName,
        Error: msg,
        Data:  ChargeDetailData{Charge: c, Payments: pays, Form: form, Today: today()},
    })
}

func (a *App) ChargeDetail(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    a.renderCharge(w, r, id, http.StatusOK, PaymentForm{}, "")
}

func (a *App) PaymentCreate(w http.ResponseWriter, r *http.Request) {
    id, ok := pathInt(r, "id")
    if !ok {
        http.NotFound(w, r)
        return
    }
    c, err := models.GetCharge(a.DB, id)
    if errors.Is(err, models.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        a.serverError(w, err)
        return
    }

    form := PaymentForm{
        Amount: strings.TrimSpace(r.PostFormValue("amount")),
        Date:   strings.TrimSpace(r.PostFormValue("paid_date")),
        Method: r.PostFormValue("method"),
        Note:   strings.TrimSpace(r.PostFormValue("note")),
    }

    amount, amountOK := parseMoney(form.Amount)
    msg := ""
    switch {
    case !amountOK:
        msg = "Amount must be a whole number above 0"
    case amount > c.Due:
        msg = fmt.Sprintf("That is more than the amount still due (%v)", c.Due)
    }
    if msg == "" {
        t, err := time.Parse(dateLayout, form.Date)
        if err != nil {
            msg = "Please enter a valid payment date"
        } else if t.Format(dateLayout) > today() {
            msg = "The payment date cannot be in the future"
        }
    }
    if msg == "" {
        switch form.Method {
        case "cash", "bank", "mobile", "other":
        default:
            msg = "Please choose a payment method"
        }
    }
    if msg == "" && utf8.RuneCountInString(form.Note) > 100 {
        msg = "The note is too long (max 100 characters)"
    }
    if msg != "" {
        a.renderCharge(w, r, id, http.StatusBadRequest, form, msg)
        return
    }

    if err := models.AddPayment(a.DB, id, amount, form.Date, form.Method, form.Note); err != nil {
        if errors.Is(err, models.ErrOverpay) {
            a.renderCharge(w, r, id, http.StatusConflict, form, "That is more than the amount still due")
            return
        }
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/charges/"+strconv.Itoa(id), http.StatusSeeOther)
}

func (a *App) PaymentDelete(w http.ResponseWriter, r *http.Request) {
    id, ok1 := pathInt(r, "id")
    pid, ok2 := pathInt(r, "pid")
    if !ok1 || !ok2 {
        http.NotFound(w, r)
        return
    }
    if err := models.DeletePayment(a.DB, id, pid); err != nil {
        a.serverError(w, err)
        return
    }
    http.Redirect(w, r, "/admin/charges/"+strconv.Itoa(id), http.StatusSeeOther)
}

// ---------- receipts and dues ----------

func (a *App) renderReceipt(w http.ResponseWriter, r *http.Request, pd *models.PaymentDetail, back string) {
    a.render(w, r, "receipt", http.StatusOK, PageData{
        Title: "Receipt " + pd.ReceiptNo(),
        Data:  ReceiptPage{R: pd, Back: back},
    })
}

func (a *App) ReceiptAdmin(w http.ResponseWriter, r *http.Request) {
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
    a.renderReceipt(w, r, pd, "/admin/charges/"+strconv.Itoa(pd.ChargeID))
}

func (a *App) DuesReport(w http.ResponseWriter, r *http.Request) {
    classes, err := models.ListClasses(a.DB)
    if err != nil {
        a.serverError(w, err)
        return
    }
    classID := 0
    if n, err := strconv.Atoi(r.URL.Query().Get("class_id")); err == nil && n > 0 {
        classID = n
    }
    rows, err := models.ListDues(a.DB, classID)
    if err != nil {
        a.serverError(w, err)
        return
    }
    var total models.Money
    for _, c := range rows {
        total += c.Due
    }
    a.render(w, r, "dues", http.StatusOK, PageData{
        Title: "Dues report",
        Data:  DuesData{Rows: rows, Classes: classes, ClassID: classID, TotalDue: total},
    })
}
