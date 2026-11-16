CREATE TABLE IF NOT EXISTS events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'event' CHECK (kind IN ('event', 'holiday', 'exam', 'meeting')),
    start_date TEXT NOT NULL,
    end_date   TEXT NOT NULL,
    details    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_events_dates ON events(start_date, end_date);
