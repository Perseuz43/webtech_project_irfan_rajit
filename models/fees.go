package models

import (
    "database/sql"
    "errors"
    "fmt"
    "time"
)

var ErrOverpay = errors.New("payment is more than the amount due")

func todayStr() string {
    return time.Now().Format("2006-01-02")
}

type FeeInput struct {
    Name    string
    Amount  Money
    DueDate string
    ClassID int // 0 = all students
}

type FeeRow struct {
    ID          int
    Name        string
    Amount      Money
    DueDate     string
    ClassID     int
    ClassName   string
    Students    int
    Expected    Money
    Collected   Money
    Outstanding Money
}

type ChargeRow struct {
    ID          int
    FeeID       int
    FeeName     string
    DueDate     string
    StudentID   int
    StudentName string
    Roll        int
    ClassName   string
    Amount      Money
    Paid        Money
    Due         Money
    Status      string // Paid, Partial or Unpaid
    Overdue     bool
}

type Payment struct {
    ID       int
    ChargeID int
    Amount   Money
    Date     string
    Method   string
    Note     string
}

func (p Payment) ReceiptNo() string {
    return fmt.Sprintf("R-%06d", p.ID)
}

func (p Payment) MethodLabel() string {
    switch p.Method {
    case "bank":
        return "Bank"
    case "mobile":
        return "Mobile banking"
    case "other":
        return "Other"
    }
    return "Cash"
}

// PaymentDetail is a payment together with everything a receipt needs.
type PaymentDetail struct {
    Payment
    FeeName      string
    StudentID    int
    StudentName  string
    Roll         int
    ClassName    string
    SectionName  string
    ChargeAmount Money
    TotalPaid    Money
    Balance      Money
}

// ---------- fees ----------

const feeSelect = `SELECT f.id, f.name, f.amount, f.due_date, COALESCE(f.class_id, 0), COALESCE(c.name, ''),
    (SELECT COUNT(*) FROM charges ch WHERE ch.fee_id = f.id),
    COALESCE((SELECT SUM(ch.amount) FROM charges ch WHERE ch.fee_id = f.id), 0),
    COALESCE((SELECT SUM(p.amount) FROM payments p JOIN charges ch ON ch.id = p.charge_id WHERE ch.fee_id = f.id), 0)
    FROM fees f LEFT JOIN classes c ON c.id = f.class_id`

const billSQL = `INSERT OR IGNORE INTO charges (fee_id, student_id, amount)
    SELECT f.id, s.id, f.amount FROM fees f
    JOIN students s ON (f.class_id IS NULL OR s.class_id = f.class_id)
    WHERE f.id = ?`

func scanFee(row rowScanner) (*FeeRow, error) {
    f := &FeeRow{}
    err := row.Scan(&f.ID, &f.Name, &f.Amount, &f.DueDate, &f.ClassID, &f.ClassName,
        &f.Students, &f.Expected, &f.Collected)
    if err != nil {
        return nil, err
    }
    f.Outstanding = f.Expected - f.Collected
    if f.ClassID == 0 {
        f.ClassName = "All students"
    }
    return f, nil
}

func ListFees(db *sql.DB) ([]FeeRow, error) {
    rows, err := db.Query(feeSelect + " ORDER BY f.due_date DESC, f.id DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []FeeRow
    for rows.Next() {
        f, err := scanFee(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *f)
    }
    return list, rows.Err()
}

func GetFee(db *sql.DB, id int) (*FeeRow, error) {
    f, err := scanFee(db.QueryRow(feeSelect+" WHERE f.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return f, nil
}

// CreateFee creates the fee and bills every student it applies to. It returns the new fee ID
// and the number of students billed.
func CreateFee(db *sql.DB, in FeeInput) (int, int64, error) {
    tx, err := db.Begin()
    if err != nil {
        return 0, 0, err
    }
    defer tx.Rollback()

    res, err := tx.Exec("INSERT INTO fees (name, amount, due_date, class_id, created_at) VALUES (?, ?, ?, ?, ?)",
        in.Name, int64(in.Amount), in.DueDate, nullInt(in.ClassID), time.Now().Format("2006-01-02 15:04"))
    if err != nil {
        return 0, 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, 0, err
    }
    billed, err := tx.Exec(billSQL, id)
    if err != nil {
        return 0, 0, err
    }
    n, err := billed.RowsAffected()
    if err != nil {
        return 0, 0, err
    }
    return int(id), n, tx.Commit()
}

// BillMissingStudents bills students who joined after the fee was created.
func BillMissingStudents(db *sql.DB, feeID int) (int64, error) {
    res, err := db.Exec(billSQL, feeID)
    if err != nil {
        return 0, err
    }
    return res.RowsAffected()
}

func DeleteFee(db *sql.DB, id int) error {
    _, err := db.Exec("DELETE FROM fees WHERE id = ?", id)
    return err
}

// ---------- charges ----------

const chargeSelect = `SELECT ch.id, ch.fee_id, f.name, f.due_date, ch.student_id, s.name, s.roll,
    COALESCE(c.name, ''), ch.amount,
    COALESCE((SELECT SUM(p.amount) FROM payments p WHERE p.charge_id = ch.id), 0)
    FROM charges ch
    JOIN fees f ON f.id = ch.fee_id
    JOIN students s ON s.id = ch.student_id
    LEFT JOIN classes c ON c.id = s.class_id`

func scanCharge(row rowScanner) (*ChargeRow, error) {
    c := &ChargeRow{}
    err := row.Scan(&c.ID, &c.FeeID, &c.FeeName, &c.DueDate, &c.StudentID, &c.StudentName,
        &c.Roll, &c.ClassName, &c.Amount, &c.Paid)
    if err != nil {
        return nil, err
    }
    c.Due = c.Amount - c.Paid
    switch {
    case c.Paid <= 0:
        c.Status = "Unpaid"
    case c.Due > 0:
        c.Status = "Partial"
    default:
        c.Status = "Paid"
    }
    c.Overdue = c.Due > 0 && c.DueDate < todayStr()
    return c, nil
}

func queryCharges(db *sql.DB, query string, args ...any) ([]ChargeRow, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []ChargeRow
    for rows.Next() {
        c, err := scanCharge(rows)
        if err != nil {
            return nil, err
        }
        list = append(list, *c)
    }
    return list, rows.Err()
}

func ListFeeCharges(db *sql.DB, feeID int) ([]ChargeRow, error) {
    return queryCharges(db, chargeSelect+" WHERE ch.fee_id = ? ORDER BY COALESCE(s.class_id, 0), s.roll", feeID)
}

func ListStudentCharges(db *sql.DB, studentID int) ([]ChargeRow, error) {
    return queryCharges(db, chargeSelect+" WHERE ch.student_id = ? ORDER BY f.due_date DESC, ch.id DESC", studentID)
}

func GetCharge(db *sql.DB, id int) (*ChargeRow, error) {
    c, err := scanCharge(db.QueryRow(chargeSelect+" WHERE ch.id = ?", id))
    if err != nil {
        return nil, notFoundIf(err)
    }
    return c, nil
}

// ListDues returns every bill that still has a balance, oldest due date first.
// classID 0 means all classes.
func ListDues(db *sql.DB, classID int) ([]ChargeRow, error) {
    all, err := queryCharges(db,
        chargeSelect+" WHERE (? = 0 OR s.class_id = ?) ORDER BY f.due_date, s.name", classID, classID)
    if err != nil {
        return nil, err
    }
    var dues []ChargeRow
    for _, c := range all {
        if c.Due > 0 {
            dues = append(dues, c)
        }
    }
    return dues, nil
}

func SummarizeCharges(list []ChargeRow) (total, paid, due Money) {
    for _, c := range list {
        total += c.Amount
        paid += c.Paid
        due += c.Due
    }
    return
}

// ---------- payments ----------

func ListPayments(db *sql.DB, chargeID int) ([]Payment, error) {
    rows, err := db.Query(`SELECT id, charge_id, amount, paid_date, method, note
        FROM payments WHERE charge_id = ? ORDER BY paid_date DESC, id DESC`, chargeID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var list []Payment
    for rows.Next() {
        var p Payment
        if err := rows.Scan(&p.ID, &p.ChargeID, &p.Amount, &p.Date, &p.Method, &p.Note); err != nil {
            return nil, err
        }
        list = append(list, p)
    }
    return list, rows.Err()
}

func GetPayment(db *sql.DB, id int) (*PaymentDetail, error) {
    d := &PaymentDetail{}
    err := db.QueryRow(`SELECT p.id, p.charge_id, p.amount, p.paid_date, p.method, p.note,
        f.name, s.id, s.name, s.roll, COALESCE(c.name, ''), COALESCE(sec.name, ''), ch.amount,
        COALESCE((SELECT SUM(x.amount) FROM payments x WHERE x.charge_id = ch.id), 0)
        FROM payments p
        JOIN charges ch ON ch.id = p.charge_id
        JOIN fees f ON f.id = ch.fee_id
        JOIN students s ON s.id = ch.student_id
        LEFT JOIN classes c ON c.id = s.class_id
        LEFT JOIN sections sec ON sec.id = s.section_id
        WHERE p.id = ?`, id).
        Scan(&d.ID, &d.ChargeID, &d.Amount, &d.Date, &d.Method, &d.Note,
            &d.FeeName, &d.StudentID, &d.StudentName, &d.Roll, &d.ClassName, &d.SectionName,
            &d.ChargeAmount, &d.TotalPaid)
    if err != nil {
        return nil, notFoundIf(err)
    }
    d.Balance = d.ChargeAmount - d.TotalPaid
    return d, nil
}

// AddPayment records a payment. It refuses a payment larger than the balance still owed.
func AddPayment(db *sql.DB, chargeID int, amount Money, date, method, note string) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var total, paid int64
    err = tx.QueryRow(`SELECT amount,
        COALESCE((SELECT SUM(amount) FROM payments WHERE charge_id = ?), 0)
        FROM charges WHERE id = ?`, chargeID, chargeID).Scan(&total, &paid)
    if err != nil {
        return notFoundIf(err)
    }
    if paid+int64(amount) > total {
        return ErrOverpay
    }

    _, err = tx.Exec(`INSERT INTO payments (charge_id, amount, paid_date, method, note, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
        chargeID, int64(amount), date, method, note, time.Now().Format("2006-01-02 15:04"))
    if err != nil {
        return err
    }
    return tx.Commit()
}

func DeletePayment(db *sql.DB, chargeID, paymentID int) error {
    _, err := db.Exec("DELETE FROM payments WHERE id = ? AND charge_id = ?", paymentID, chargeID)
    return err
}
