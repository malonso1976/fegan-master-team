package domain

import (
	"reflect"
	"testing"
)

func TestConversionTableMatchesSuppliedImage(t *testing.T) {
	events := []string{"50L", "100L", "200L", "400L", "800L", "1500L", "50E", "100E", "200E", "50B", "100B", "200B", "50M", "100M", "200M", "200S", "400S", "4x50L", "4x100L", "4x200L", "4x50S", "4x100S"}
	expected := map[string][]int{
		"M": {60, 160, 350, 670, 1290, 2350, 150, 290, 550, 70, 240, 540, 40, 110, 290, 370, 880, 240, 640, 1400, 320, 800},
		"F": {40, 110, 250, 620, 1100, 2060, 90, 210, 390, 50, 140, 440, 30, 70, 170, 290, 630, 160, 440, 1000, 210, 530},
	}
	for gender, offsets := range expected {
		if len(conversions.PoolOffsets[gender]) != len(events) {
			t.Fatal("incomplete table", gender)
		}
		for i, event := range events {
			m := Mark{Event: event, Gender: gender, Pool: 25, Chrono: "E", Centiseconds: 10000}
			cs, ok := ConvertedTime(m, 50)
			if !ok || cs != 10000+offsets[i] {
				t.Fatal(gender, event, cs, ok)
			}
			m.Pool = 50
			m.Centiseconds = cs
			back, ok := ConvertedTime(m, 25)
			if !ok || back != 10000 {
				t.Fatal("inverse", gender, event, back, ok)
			}
		}
	}
	for event, want := range map[string]int{"4x50L": 200, "4x100L": 540, "4x200L": 1200, "4x50S": 270, "4x100S": 670} {
		cs, ok := ConvertedTime(Mark{Event: event, Gender: "X", Pool: 25, Chrono: "E", Centiseconds: 10000}, 50)
		if !ok || cs != 10000+want {
			t.Fatal(event, cs, ok)
		}
	}
	for _, event := range []string{"100S", "2000L", "3000L", "5000L"} {
		if _, ok := ConvertedTime(Mark{Event: event, Gender: "M", Pool: 25, Chrono: "E", Centiseconds: 10000}, 50); ok {
			t.Fatal("invented offset", event)
		}
	}
}
func TestManualTimingCorrectionAndInvalidConversions(t *testing.T) {
	for _, tc := range []struct {
		event, gender      string
		from, to, cs, want int
	}{
		{"50L", "M", 25, 50, 3000, 3089}, {"50L", "F", 25, 50, 3000, 3069},
		{"100L", "M", 25, 50, 6000, 6179}, {"100L", "F", 50, 25, 6000, 5909},
		{"100S", "M", 25, 25, 8000, 8019}, {"4x50L", "M", 25, 50, 10000, 10259},
	} {
		m := Mark{Event: tc.event, Gender: tc.gender, Pool: tc.from, Chrono: "M", Centiseconds: tc.cs}
		if got, ok := ConvertedTime(m, tc.to); !ok || got != tc.want {
			t.Fatal(tc, got, ok)
		}
	}
	for _, m := range []Mark{{Event: "50L", Gender: "M", Pool: 25, Chrono: "E", Centiseconds: 0}, {Event: "50L", Gender: "M", Pool: 50, Chrono: "E", Centiseconds: 10}, {Event: "50L", Gender: "M", Pool: 20, Chrono: "E", Centiseconds: 3000}, {Event: "50L", Gender: "M", Pool: 25, Chrono: "?", Centiseconds: 3000}} {
		if _, ok := ConvertedTime(m, 25); ok {
			t.Fatal("invalid conversion", m)
		}
	}
}
func TestConvertedOpportunitiesCombinePoolsAndTimingsWithoutChangingSources(t *testing.T) {
	d := Data{Swimmers: []Swimmer{{License: "a", Gender: "M", BirthYear: 1986}, {License: "b", Gender: "M", BirthYear: 1985}, {License: "c", Gender: "M", BirthYear: 1984}}}
	add := func(license, event, date, chrono string, pool, cs int, verified bool) {
		m := mark(license, event, cs)
		m.Date = date
		m.Pool = pool
		m.Chrono = chrono
		m.Verified = verified
		d.Marks = append(d.Marks, m)
	}
	add("a", "50L", "2015-01-01", "M", 25, 2900, true) // 29.89 beats E50 30.00
	add("a", "50L", "2026-01-01", "E", 50, 3000, true)
	add("a", "50L", "2026-01-01", "E", 50, 2000, false) // unverified excluded
	add("a", "50L", "2028-01-01", "E", 50, 1000, true)  // future excluded
	add("b", "50L", "2026-01-01", "M", 25, 2800, true)  // 28.89, selected once
	add("b", "50L", "2026-01-02", "E", 50, 2900, true)
	add("b", "50L", "2020-01-01", "E", 50, 2000, true) // old rival excluded
	add("c", "50L", "2026-01-01", "E", 50, 2989, true) // tie with own after conversion
	add("a", "100S", "2020-01-01", "M", 25, 8000, true)
	add("b", "100S", "2026-01-01", "E", 25, 7900, true)
	add("a", "3000L", "2020-01-01", "E", 25, 300000, true)
	add("a", "3000L", "2020-01-01", "E", 50, 310000, true)
	add("b", "3000L", "2026-01-01", "M", 50, 300000, true)
	before := append([]Mark(nil), d.Marks...)
	rows, err := Recommend(d, "a", Filter{From: "2024-10-28", To: "2026-10-28", SeasonStart: 2026, VerifiedOnly: true})
	if err != nil || len(rows) != 4 {
		t.Fatal(rows, err)
	}
	for _, r := range rows {
		if r.Chrono != "E" || r.Conversion == nil {
			t.Fatal(r)
		}
		switch r.Event {
		case "50L":
			if r.Time != 2989 || r.OwnMark.Centiseconds != 2900 || r.OwnMark.Chrono != "M" || r.Pool != 50 || r.Rank != 2 || r.Rivals != 2 || r.GapToFirst != 100 || len(r.Warnings) != 0 {
				t.Fatal(r)
			}
			rival := r.Details[0]
			if rival.Mark.Chrono != "M" || rival.Mark.Pool != 25 || rival.Time != 2889 || rival.Mark.Centiseconds != 2800 {
				t.Fatal(rival)
			}
		case "100S":
			if r.Pool != 25 || r.Time != 8019 || len(r.Warnings) != 1 {
				t.Fatal(r)
			}
		case "3000L":
			if len(r.Warnings) != 1 {
				t.Fatal(r)
			}
		}
	}
	if !reflect.DeepEqual(before, d.Marks) {
		t.Fatal("mutated original data")
	}
}
func TestConversionTiePrefersOriginalElectronicReference(t *testing.T) {
	a := convertedReference{mark: Mark{Chrono: "M", Date: "2026-01-01"}, time: 3000}
	b := convertedReference{mark: Mark{Chrono: "E", Date: "2015-01-01"}, time: 3000}
	if !b.better(a) || a.better(b) {
		t.Fatal("tie must prefer E")
	}
}
