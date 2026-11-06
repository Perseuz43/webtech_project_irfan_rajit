package models

import "testing"

func TestOrdinal(t *testing.T) {
    want := map[int]string{
        1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 10: "10th", 11: "11th", 12: "12th", 13: "13th",
        21: "21st", 22: "22nd", 23: "23rd", 101: "101st", 111: "111th", 112: "112th", 113: "113th",
    }
    for n, s := range want {
        if got := Ordinal(n); got != s {
            t.Errorf("Ordinal(%d) = %q, want %q", n, got, s)
        }
    }
}

func TestRankText(t *testing.T) {
    if got := (&ReportCard{}).RankText(); got != "" {
        t.Errorf("no rank should give an empty text, got %q", got)
    }
    rc := &ReportCard{HasRank: true, Rank: 2, RankOf: 18}
    if got := rc.RankText(); got != "2nd of 18" {
        t.Errorf("got %q", got)
    }
}
