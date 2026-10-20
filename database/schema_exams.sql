CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS grading_scales (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT NOT NULL UNIQUE,
    pass_percent REAL NOT NULL CHECK (pass_percent >= 0 AND pass_percent <= 100),
    use_gpa      INTEGER NOT NULL DEFAULT 0 CHECK (use_gpa IN (0, 1)),
    is_preset    INTEGER NOT NULL DEFAULT 0 CHECK (is_preset IN (0, 1))
);

CREATE TABLE IF NOT EXISTS grade_bands (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    scale_id    INTEGER NOT NULL REFERENCES grading_scales(id) ON DELETE CASCADE,
    min_percent REAL NOT NULL CHECK (min_percent >= 0 AND min_percent <= 100),
    grade       TEXT NOT NULL,
    point       REAL NOT NULL DEFAULT 0,
    UNIQUE (scale_id, min_percent)
);

CREATE TABLE IF NOT EXISTS exams (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id  INTEGER NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    exam_date TEXT NOT NULL,
    published INTEGER NOT NULL DEFAULT 0 CHECK (published IN (0, 1)),
    UNIQUE (class_id, name)
);

CREATE TABLE IF NOT EXISTS exam_subjects (
    exam_id    INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    full_marks REAL NOT NULL DEFAULT 100 CHECK (full_marks > 0),
    PRIMARY KEY (exam_id, subject_id)
);

CREATE TABLE IF NOT EXISTS marks (
    exam_id    INTEGER NOT NULL,
    subject_id INTEGER NOT NULL,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    marks      REAL NOT NULL CHECK (marks >= 0),
    PRIMARY KEY (exam_id, subject_id, student_id),
    FOREIGN KEY (exam_id, subject_id) REFERENCES exam_subjects(exam_id, subject_id) ON DELETE CASCADE
);
