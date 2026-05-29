package cmd

import "testing"

func TestResolveYearMonth(t *testing.T) {
	cases := []struct {
		name      string
		date      string
		year      int
		month     int
		wantY     int
		wantM     int
		wantError bool
	}{
		{name: "date wins", date: "2026-05-04", wantY: 2026, wantM: 5},
		{name: "year+month", year: 2026, month: 5, wantY: 2026, wantM: 5},
		{name: "year only is an error", year: 2025, wantError: true},
		{name: "month only is an error", month: 7, wantError: true},
		{name: "bad date", date: "2026/05/04", wantError: true},
		{name: "month out of range", year: 2026, month: 13, wantError: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			y, m, err := resolveYearMonth(c.date, c.year, c.month)
			if c.wantError {
				if err == nil {
					t.Fatalf("expected error, got (%d, %d)", y, m)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if y != c.wantY || m != c.wantM {
				t.Fatalf("got (%d, %d), want (%d, %d)", y, m, c.wantY, c.wantM)
			}
		})
	}

	// No flags at all → current month, no error.
	if _, m, err := resolveYearMonth("", 0, 0); err != nil || m < 1 || m > 12 {
		t.Fatalf("default month: m=%d err=%v", m, err)
	}
}
