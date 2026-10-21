CREATE TABLE IF NOT EXISTS notices (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    audience   TEXT NOT NULL DEFAULT 'all' CHECK (audience IN ('all', 'teacher', 'student', 'parent', 'class')),
    class_id   INTEGER REFERENCES classes(id) ON DELETE CASCADE,
    pinned     INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    created_at TEXT NOT NULL,
    CHECK ((audience = 'class' AND class_id IS NOT NULL) OR (audience <> 'class' AND class_id IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_notices_created ON notices(created_at);
