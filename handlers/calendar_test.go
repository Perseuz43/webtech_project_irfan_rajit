package handlers

import (
    "testing"
    "time"

    "github.com/Perseuz43/webtech_project_irfan_rajit/models"
)

func TestBuildMonthGrid(t *testing.T) {
    today := time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC)
    cases := []struct {
        year       int
        month      time.Month
        wantWeeks  int
        wantFirst  string
        wantLast   string
    }{
        {2026, time.November, 5, "2026-11-01", "2026-12-05"},  // starts on a Sunday
        {2026, time.February, 4, "2026-02-01", "2026-02-28"},  // exactly four weeks
        {2027, time.January, 6, "2026-12-27", "2027-02-06"},   // starts on a Friday, ends on a Sunday
        {2028, time.February, 5, "2028-01-30", "2028-03-04"},  // leap year
    }
    for _, c := range cases {
        weeks := buildMonthGrid(c.year, c.month, today, nil)
        if len(weeks) != c.wantWeeks {
            t.Errorf("%d-%02d: got %d weeks, want %d", c.year, c.month, len(weeks), c.wantWeeks)
            continue
        }
        for i, w := range weeks {
            if len(w) != 7 {
                t.Errorf("%d-%02d week %d has %d days", c.year, c.month, i, len(w))
            }
        }
        if got := weeks[0][0].Date; got != c.wantFirst {
            t.Errorf("%d-%02d: first day %s, want %s", c.year, c.month, got, c.wantFirst)
        }
        if got := weeks[len(weeks)-1][6].Date; got != c.wantLast {
            t.Errorf("%d-%02d: last day %s, want %s", c.year, c.month, got, c.wantLast)
        }
    }
}

func TestMonthGridMarksTodayAndEvents(t *testing.T) {
    today := time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC)
    events := []models.Event{
        {Title: "Eid break", Kind: "holiday", StartDate: "2026-11-10", EndDate: "2026-11-12"},
        {Title: "A", StartDate: "2026-11-11", EndDate: "2026-11-11"},
        {Title: "B", StartDate: "2026-11-11", EndDate: "2026-11-11"},
        {Title: "C", StartDate: "2026-11-11", EndDate: "2026-11-11"},
        {Title: "D", StartDate: "2026-11-11", EndDate: "2026-11-11"},
    }
    weeks := buildMonthGrid(2026, time.November, today, events)

    var todays, eidDays int
    var crowded CalendarDay
    for _, w := range weeks {
        for _, d := range w {
            if d.Today {
                todays++
            }
            for _, e := range d.Events {
                if e.Title == "Eid break" {
                    eidDays++
                }
            }
            if d.Date == "2026-11-11" {
                crowded = d
            }
        }
    }
    if todays != 1 {
        t.Errorf("exactly one day should be today, got %d", todays)
    }
    if eidDays != 3 {
        t.Errorf("a 3-day event should appear on 3 days, got %d", eidDays)
    }
    if len(crowded.Events) != maxCellEvents || crowded.More != 2 {
        t.Errorf("11 Nov: %d shown, %d more; want %d shown, 2 more", len(crowded.Events), crowded.More, maxCellEvents)
    }
}
