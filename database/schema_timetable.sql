CREATE TABLE IF NOT EXISTS timetable_slots (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id   INTEGER NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    day        INTEGER NOT NULL CHECK (day BETWEEN 0 AND 6),      -- 0 = Sunday ... 6 = Saturday
    period     INTEGER NOT NULL CHECK (period BETWEEN 1 AND 12),
    subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    teacher_id INTEGER REFERENCES teachers(id) ON DELETE SET NULL,
    UNIQUE (class_id, day, period)
);

CREATE INDEX IF NOT EXISTS idx_timetable_teacher ON timetable_slots(teacher_id, day, period);

CREATE TABLE IF NOT EXISTS timetable_periods (
    period     INTEGER PRIMARY KEY CHECK (period BETWEEN 1 AND 12),
    start_time TEXT NOT NULL,
    end_time   TEXT NOT NULL
);
