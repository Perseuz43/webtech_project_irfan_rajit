CREATE TABLE IF NOT EXISTS report_remarks (
    exam_id    INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    remark     TEXT NOT NULL,
    PRIMARY KEY (exam_id, student_id)
);
