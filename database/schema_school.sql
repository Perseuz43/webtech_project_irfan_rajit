CREATE TABLE IF NOT EXISTS school_logo (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    content_type TEXT NOT NULL,
    data         BLOB NOT NULL,
    version      TEXT NOT NULL
);
