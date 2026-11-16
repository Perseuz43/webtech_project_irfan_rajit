package models

import "testing"

func TestAcademicYearContains(t *testing.T) {
    y := AcademicYear{StartDate: "2026-01-01", EndDate: "2026-12-31"}
    for date, want := range map[string]bool{
        "2026-01-01": true, "2026-06-15": true, "2026-12-31": true,
        "2025-12-31": false, "2027-01-01": false,
    } {
        if got := y.Contains(date); got != want {
            t.Errorf("Contains(%s) = %v, want %v", date, got, want)
        }
    }
}

func TestDefaultYear(t *testing.T) {
    years := []AcademicYear{
        {ID: 2, Name: "2026", StartDate: "2026-01-01", EndDate: "2026-12-31"},
        {ID: 1, Name: "2025", StartDate: "2025-01-01", EndDate: "2025-12-31", IsCurrent: true},
    }
    if y := DefaultYear(years, "2026-05-05"); y == nil || y.ID != 1 {
        t.Errorf("the year marked current should win, got %+v", y)
    }
    years[1].IsCurrent = false
    if y := DefaultYear(years, "2026-05-05"); y == nil || y.ID != 2 {
        t.Errorf("without a current year, the one containing today should win, got %+v", y)
    }
    if y := DefaultYear(years, "2030-01-01"); y != nil {
        t.Errorf("no year fits, want nil, got %+v", y)
    }
    if y := DefaultYear(nil, "2026-01-01"); y != nil {
        t.Errorf("no years at all, want nil")
    }
}
