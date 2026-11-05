CREATE TABLE IF NOT EXISTS student_details (
    student_id INTEGER PRIMARY KEY REFERENCES students(id) ON DELETE CASCADE,
    dob        TEXT NOT NULL DEFAULT ''
);
