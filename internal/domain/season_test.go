package domain

import (
	"testing"
	"time"
)

func TestSeasonBoundaryAndOfficialExamples(t *testing.T) {
	for date, want := range map[string]int{"2026-09-30": 2025, "2026-10-01": 2026, "2027-01-01": 2026, "2027-09-30": 2026, "2027-10-01": 2027} {
		d, _ := time.Parse("2006-01-02", date)
		if got := SeasonStartOn(d); got != want {
			t.Fatalf("%s: %d", date, got)
		}
	}
	for birth, want := range map[int]int{2007: 20, 2003: 24, 1987: 40, 1983: 44, 1927: 100, 1920: 107} {
		if age := SeasonAge(2026, birth); age != want {
			t.Fatal(birth, age)
		}
	}
	if l, h := IndividualGroup(107); l != 100 || h != 0 {
		t.Fatal(l, h)
	}
	if l, h := IndividualGroup(44); l != 40 || h != 44 {
		t.Fatal(l, h)
	}
	if l, _ := IndividualGroup(19); l != 0 {
		t.Fatal("underage accepted")
	}
	for age, want := range map[int]int{79: 0, 80: 80, 99: 80, 100: 100, 119: 100, 120: 120, 159: 120, 160: 160, 319: 280, 320: 320, 420: 320} {
		if got := NaturalRelayCategory(age); got != want {
			t.Fatal(age, got)
		}
	}
}
