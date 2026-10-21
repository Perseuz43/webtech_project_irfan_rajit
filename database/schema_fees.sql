CREATE TABLE IF NOT EXISTS fees (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    amount     INTEGER NOT NULL CHECK (amount > 0),
    due_date   TEXT NOT NULL,
    class_id   INTEGER REFERENCES classes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS charges (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    fee_id     INTEGER NOT NULL REFERENCES fees(id) ON DELETE CASCADE,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    amount     INTEGER NOT NULL CHECK (amount > 0),
    UNIQUE (fee_id, student_id)
);

CREATE TABLE IF NOT EXISTS payments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    charge_id  INTEGER NOT NULL REFERENCES charges(id) ON DELETE CASCADE,
    amount     INTEGER NOT NULL CHECK (amount > 0),
    paid_date  TEXT NOT NULL,
    method     TEXT NOT NULL CHECK (method IN ('cash', 'bank', 'mobile', 'other')),
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_charges_student ON charges(student_id);
CREATE INDEX IF NOT EXISTS idx_payments_charge ON payments(charge_id);
