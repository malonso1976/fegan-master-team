package domain

import "testing"

func TestOpportunityUsesHistoricalOwnBestAndRecentMatchingRivals(t *testing.T) {
	d := Data{Swimmers: []Swimmer{{License: "a", BirthYear: 1986, Gender: "M"}, {License: "b", BirthYear: 1985, Gender: "M"}, {License: "c", BirthYear: 1984, Gender: "M"}}}
	add := func(l, event, date, chrono string, pool, cs int) {
		m := mark(l, event, cs)
		m.Date = date
		m.Chrono = chrono
		m.Pool = pool
		d.Marks = append(d.Marks, m)
	}
	add("a", "50L", "2015-01-01", "M", 25, 2600)
	add("a", "50L", "2026-01-01", "E", 25, 2800)
	add("a", "50L", "2026-02-01", "M", 25, 3000)
	add("a", "50L", "2028-01-01", "M", 25, 1000)  // beyond comparison date
	add("a", "50L", "2026-02-01", "E", 50, 1500)  // other pool
	add("a", "100L", "2012-01-01", "E", 25, 6000) // no own recent reference needed
	add("b", "50L", "2015-01-01", "M", 25, 2400)  // old rival excluded
	add("b", "50L", "2026-01-01", "E", 25, 2000)  // faster but timing incompatible
	add("b", "50L", "2026-02-01", "M", 25, 2700)
	add("b", "50L", "2026-03-01", "M", 25, 2650) // one rival, best recent reference
	add("c", "50L", "2026-02-01", "E", 25, 2100) // no comparable reference
	add("b", "100L", "2026-03-01", "E", 25, 5900)
	f := Filter{From: "2024-10-27", To: "2026-10-27", SeasonStart: 2026, Pool: 25, VerifiedOnly: true}
	rows, e := Recommend(d, "a", f)
	if e != nil || len(rows) != 2 {
		t.Fatal(rows, e)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if seen[r.Event] {
			t.Fatal("duplicate event", r)
		}
		seen[r.Event] = true
		if r.Event == "50L" && (r.Time != 2600 || r.Chrono != "M" || r.OwnMark.Date != "2015-01-01" || r.Rivals != 1 || r.Rank != 1 || r.Details[0].Mark.Centiseconds != 2650 || r.Details[0].Swimmer.BirthYear != 1985) {
			t.Fatal(r)
		}
		if r.Event == "100L" && (r.Time != 6000 || r.OwnMark.Date != "2012-01-01" || r.Rank != 2) {
			t.Fatal(r)
		}
	}
}
func TestOpportunityTiePrefersElectronicAndIncludesBoundaryDates(t *testing.T) {
	d := Data{Swimmers: []Swimmer{{License: "a", BirthYear: 1986, Gender: "M"}, {License: "b", BirthYear: 1985, Gender: "M"}}}
	for _, chrono := range []string{"M", "E"} {
		m := mark("a", "50L", 3000)
		m.Chrono = chrono
		m.Date = "2010-01-01"
		d.Marks = append(d.Marks, m)
	}
	r := mark("b", "50L", 2900)
	r.Date = "2024-10-27"
	d.Marks = append(d.Marks, r)
	f := Filter{From: "2024-10-27", To: "2026-10-27", SeasonStart: 2026, Pool: 25, VerifiedOnly: true}
	for i := 0; i < 2; i++ {
		rows, e := Recommend(d, "a", f)
		if e != nil || len(rows) != 1 || rows[0].Chrono != "E" || rows[0].Rivals != 1 {
			t.Fatal(rows, e)
		}
		d.Marks[0], d.Marks[1] = d.Marks[1], d.Marks[0]
	}
}
