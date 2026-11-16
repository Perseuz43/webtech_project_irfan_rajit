CREATE TABLE IF NOT EXISTS academic_years (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    start_date TEXT NOT NULL,
    end_date   TEXT NOT NULL,
    is_current INTEGER NOT NULL DEFAULT 0 CHECK (is_current IN (0, 1))
);

-- at most one year can be the current one
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_year ON academic_years(is_current) WHERE is_current = 1;
