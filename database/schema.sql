CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'teacher', 'student')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS classes (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS sections (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    name     TEXT NOT NULL,
    UNIQUE (class_id, name)
);

CREATE TABLE IF NOT EXISTS subjects (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    name     TEXT NOT NULL,
    UNIQUE (class_id, name)
);

CREATE TABLE IF NOT EXISTS students (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    roll          INTEGER NOT NULL,
    class_id      INTEGER REFERENCES classes(id) ON DELETE SET NULL,
    section_id    INTEGER REFERENCES sections(id) ON DELETE SET NULL,
    guardian_name TEXT NOT NULL DEFAULT '',
    phone         TEXT NOT NULL DEFAULT '',
    UNIQUE (class_id, roll)
);

CREATE TABLE IF NOT EXISTS teachers (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    phone         TEXT NOT NULL DEFAULT '',
    qualification TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS teacher_subjects (
    teacher_id INTEGER NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    PRIMARY KEY (teacher_id, subject_id)
);

CREATE TABLE IF NOT EXISTS attendance (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    date       TEXT NOT NULL,
    status     TEXT NOT NULL CHECK (status IN ('present', 'late', 'absent')),
    UNIQUE (student_id, date)
);

CREATE INDEX IF NOT EXISTS idx_attendance_date ON attendance(date);
