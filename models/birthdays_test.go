package models

import (
    "testing"
    "time"
)

func day(y int, m time.Month, d int) time.Time {
    return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestNextBirthday(t *testing.T) {
    cases := []struct {
        name        string
        dob, today  time.Time
        wantNext    time.Time
        wantTurning int
        wantDays    int
    }{
        {"later this year", day(2012, time.December, 5), day(2026, time.November, 4), day(2026, time.December, 5), 14, 31},
        {"today counts", day(2012, time.November, 4), day(2026, time.November, 4), day(2026, time.November, 4), 14, 0},
        {"already passed, so next year", day(2012, time.March, 1), day(2026, time.November, 4), day(2027, time.March, 1), 15, 117},
        {"year end rollover", day(2010, time.January, 2), day(2026, time.December, 30), day(2027, time.January, 2), 17, 3},
        {"29 Feb in a normal year is 28 Feb", day(2012, time.February, 29), day(2026, time.January, 1), day(2026, time.February, 28), 14, 58},
        {"29 Feb in a leap year stays 29 Feb", day(2012, time.February, 29), day(2028, time.January, 1), day(2028, time.February, 29), 16, 59},
    }
    for _, c := range cases {
        next, turning := nextBirthday(c.dob, c.today)
        if !next.Equal(c.wantNext) || turning != c.wantTurning {
            t.Errorf("%s: got %s turning %d, want %s turning %d", c.name, next.Format("2006-01-02"), turning, c.wantNext.Format("2006-01-02"), c.wantTurning)
        }
        if got := daysBetween(c.today, next); got != c.wantDays {
            t.Errorf("%s: got %d days, want %d", c.name, got, c.wantDays)
        }
    }
}

func TestValidDOB(t *testing.T) {
    today := day(2026, time.November, 4)
    good := []string{"2012-05-17", "2026-11-04", "1999-02-28", "2012-02-29"}
    bad := []string{"", "2026-11-05", "2013-02-29", "17/05/2012", "1900-01-01", "2012-13-01", "abc"}
    for _, s := range good {
        if !ValidDOB(s, today) {
            t.Errorf("%q should be valid", s)
        }
    }
    for _, s := range bad {
        if ValidDOB(s, today) {
            t.Errorf("%q should be rejected", s)
        }
    }
}
