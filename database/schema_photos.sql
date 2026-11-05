CREATE TABLE IF NOT EXISTS student_photos (
    student_id   INTEGER PRIMARY KEY REFERENCES students(id) ON DELETE CASCADE,
    content_type TEXT NOT NULL,
    data         BLOB NOT NULL,
    version      TEXT NOT NULL
);
