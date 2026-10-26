package fegan

import "testing"

func TestExpandEmptyFiltersWithoutConversions(t *testing.T) {
	q := Query{ClubID: "001", Event: "all", From: "2025-01-01", To: "2026-01-01"}
	rows, e := ExpandQuery(q)
	if e != nil || len(rows) != 8 {
		t.Fatal(rows, e)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Pool != 25 && r.Pool != 50 {
			t.Fatal(r)
		}
		seen[r.Gender+r.Chrono+string(rune(r.Pool))] = true
	}
	if len(seen) != 8 {
		t.Fatal(rows)
	}
	q.Gender = "M"
	rows, e = ExpandQuery(q)
	if e != nil || len(rows) != 4 {
		t.Fatal(rows, e)
	}
	q.Gender = "X"
	if _, e = ExpandQuery(q); e == nil {
		t.Fatal("invalid gender accepted")
	}
}
