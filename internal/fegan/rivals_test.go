package fegan

import (
	"fmt"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"testing"
	"time"
)

func TestRivalQueriesSharedGroupsAndRollingWindow(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	d := domain.Data{Swimmers: []domain.Swimmer{{License: "a", BirthYear: 1987, Gender: "M", ClubID: "001", Member: true}, {License: "b", BirthYear: 1983, Gender: "M", ClubID: "001", Member: true}, {License: "c", BirthYear: 1982, Gender: "M", ClubID: "001", Member: true}, {License: "rival", BirthYear: 1987, Gender: "M", ClubID: "002"}}}
	for _, l := range []string{"a", "b", "c", "rival"} {
		d.Marks = append(d.Marks, domain.Mark{License: l, Event: "50L", Pool: 25, Chrono: "E", Date: "2025-01-01"})
	}
	old := d.Marks[0]
	old.Event = "100L"
	old.Date = "2024-10-25"
	d.Marks = append(d.Marks, old)
	queries := RivalQueries(d, "001", 2026, now)
	if len(queries) != 3 {
		t.Fatal(queries)
	}
	for _, q := range queries {
		if !q.TopTen || q.From != "2024-10-26" || q.To != "2026-10-26" || (q.Event != "50L" && q.Event != "100L") {
			t.Fatal(q)
		}
		if q.BirthTo == 1987 && q.BirthFrom != 1983 {
			t.Fatal(q)
		}
	}
}
func TestTopTenUsesDistinctLicenses(t *testing.T) {
	d := domain.Data{}
	for i := 0; i < 12; i++ {
		l := fmt.Sprint(i)
		d.Swimmers = append(d.Swimmers, domain.Swimmer{License: l})
		d.Marks = append(d.Marks, domain.Mark{License: l, Centiseconds: 3000 + i}, domain.Mark{License: l, Centiseconds: 4000 + i})
	}
	got := topTen(d)
	if len(got.Marks) != 10 || len(got.Swimmers) != 10 || got.Marks[9].Centiseconds != 3009 {
		t.Fatal(got)
	}
}
