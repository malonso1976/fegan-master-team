package domain

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func adviceSquad() Data {
	d := Data{}
	for i := 0; i < 4; i++ {
		l := fmt.Sprintf("s%d", i)
		d.Swimmers = append(d.Swimmers, Swimmer{License: l, Name: l, BirthYear: 1982, Gender: "M", ClubID: "001", Member: true, Available: true})
		d.Marks = append(d.Marks, mark(l, "50L", 3000))
	}
	return d
}
func adviceRequest() RelayRequest {
	return RelayRequest{Filter: Filter{From: "2023-01-01", To: "2026-12-31", SeasonStart: 2026, Pool: 25, Chrono: "E", VerifiedOnly: true}, ClubID: "001", Event: "4x50L", Gender: "M", Competition: "championship", Limit: 2}
}
func TestLowerCategoriesAndBestMedalReference(t *testing.T) {
	d := adviceSquad()
	q := adviceRequest()
	for category, times := range map[int][]int{120: {11500, 11700, 11900}, 160: {13000, 14000, 15000}} {
		for i, cs := range times {
			m := mark("", "4x50L", cs)
			m.ClubID = fmt.Sprintf("r%d", i)
			m.RelayCategory = category
			m.AgeTotal = 220
			d.Marks = append(d.Marks, m)
		}
	}
	advice, e := RecommendRelayCategories(context.Background(), d, q)
	if e != nil {
		t.Fatal(e)
	}
	if advice.RecommendedCategory != 160 || advice.AgeAt != "31/12/2027" {
		t.Fatalf("%+v", advice)
	}
	for _, g := range advice.Groups {
		if g.Category <= 160 {
			if len(g.Results) != 1 || g.Results[0].AgeTotal != 180 {
				t.Fatalf("missing lower category: %+v", g)
			}
		} else if len(g.Results) > 0 {
			t.Fatal("entered higher category")
		}
	}
	if advice.Groups[2].Results[0].Rank != 4 || *advice.Groups[2].Results[0].GapToPodium != 100 {
		t.Fatal("wrong bronze reference")
	}
	q.Category = 120
	advice, e = RecommendRelayCategories(context.Background(), d, q)
	if e != nil || len(advice.Groups) != 1 || advice.Groups[0].Category != 120 {
		t.Fatal(advice, e)
	}
}
func TestPremasterRestrictionAndNoInventedRecommendation(t *testing.T) {
	d := adviceSquad()
	d.Swimmers[0].BirthYear = 2004 // 23 at 31/12/2027: +80 only, even with a total above 99.
	q := adviceRequest()
	advice, e := RecommendRelayCategories(context.Background(), d, q)
	if e != nil {
		t.Fatal(e)
	}
	if advice.RecommendedCategory != 0 {
		t.Fatal("invented recommendation without rivals")
	}
	for _, g := range advice.Groups {
		if g.Category == 80 {
			if len(g.Results) != 1 {
				t.Fatal("premaster not eligible +80")
			}
		} else if len(g.Results) > 0 {
			t.Fatalf("premaster allowed +%d", g.Category)
		}
	}
	d.Swimmers[0].BirthYear = 2008 // under 20 at reference date
	advice, e = RecommendRelayCategories(context.Background(), d, q)
	if e != nil {
		t.Fatal(e)
	}
	for _, g := range advice.Groups {
		if len(g.Results) > 0 {
			t.Fatal("underage swimmer used")
		}
	}
	q.Event = "4x100L"
	if _, e = RecommendRelayCategories(context.Background(), d, q); e == nil {
		t.Fatal("event outside programme accepted")
	}
}
func TestChampionshipHistoricalDateAndVerifiedConstraints(t *testing.T) {
	d := adviceSquad()
	for i := range d.Marks {
		d.Marks[i].Date = "2023-08-31"
	}
	q := adviceRequest()
	q.VerifiedOnly = false
	advice, e := RecommendRelayCategories(context.Background(), d, q)
	if e != nil {
		t.Fatal(e)
	}
	for _, g := range advice.Groups {
		if len(g.Results) > 0 {
			t.Fatal("pre-accreditation reference accepted")
		}
	}
	for i := range d.Marks {
		d.Marks[i].Date = "2026-01-01"
		d.Marks[i].Verified = false
	}
	advice, e = RecommendRelayCategories(context.Background(), d, q)
	if e != nil {
		t.Fatal(e)
	}
	for _, g := range advice.Groups {
		if len(g.Results) > 0 {
			t.Fatal("unverified championship reference accepted")
		}
	}
	q.Competition = "league"
	advice, e = RecommendRelayCategories(context.Background(), d, q)
	if e != nil || len(advice.Groups[0].Results) != 1 {
		t.Fatal("league approximate references rejected", e)
	}
}

func TestRuleTypesDoNotDependOnSeasonOrCircularURL(t *testing.T) {
	for _, season := range []int{2027, 2032} {
		d := adviceSquad()
		q := adviceRequest()
		q.SeasonStart = season
		q.To = fmt.Sprintf("%d-09-30", season+1)
		for _, rules := range []string{ChampionshipRules, LeagueRules, ""} {
			q.Competition = rules
			advice, e := RecommendRelayCategories(context.Background(), d, q)
			if e != nil || advice.Rules != rules || len(advice.Groups[0].Results) == 0 {
				t.Fatal(season, advice, e)
			}
			for _, warning := range advice.Warnings {
				if strings.Contains(warning, "2627") || strings.Contains(warning, "2026/27") || strings.Contains(warning, "https://") {
					t.Fatal("season-specific warning", warning)
				}
			}
		}
	}
	d := adviceSquad()
	q := adviceRequest()
	for i := range d.Marks {
		d.Marks[i].Date = ChampionshipMarksSince
	}
	advice, e := RecommendRelayCategories(context.Background(), d, q)
	if e != nil || len(advice.Groups[0].Results) != 1 {
		t.Fatal("historic boundary excluded", advice, e)
	}
}

func TestRelayRivalsRemainVisibleWithoutElectronicSquad(t *testing.T) {
	for _, gender := range []string{"M", "F"} {
		t.Run(gender, func(t *testing.T) {
			d := adviceSquad()
			for i := range d.Swimmers {
				d.Swimmers[i].Gender = gender
				d.Marks[i].Gender = gender
			}
			d.Marks[3].Chrono = "M" // Four swimmers overall, only three with electronic references.
			q := adviceRequest()
			q.Gender = gender
			q.Category = 120
			for _, id := range []string{"002", "003", "004"} {
				d.Clubs = append(d.Clubs, Club{ID: id, Name: "Club " + id})
				m := mark("", "4x50L", 11500)
				m.Gender = gender
				m.RelayCategory = 120
				m.ClubID = id
				m.ID = id
				d.Marks = append(d.Marks, m)
				older := m
				older.Centiseconds = 11700
				older.Date = "2025-01-01"
				d.Marks = append(d.Marks, older)
			}
			original := d.Marks[len(d.Marks)-2]
			for _, kind := range []string{"gender", "pool", "chrono", "date", "category", "unverified", "own"} {
				m := original
				m.ClubID = kind
				m.Centiseconds = 10000
				switch kind {
				case "gender":
					if gender == "M" {
						m.Gender = "F"
					} else {
						m.Gender = "M"
					}
				case "pool":
					m.Pool = 50
				case "chrono":
					m.Chrono = "M"
				case "date":
					m.Date = "2020-01-01"
				case "category":
					m.RelayCategory = 160
				case "unverified":
					m.Verified = false
				case "own":
					m.ClubID = q.ClubID
				}
				d.Marks = append(d.Marks, m)
			}
			advice, err := RecommendRelayCategories(context.Background(), d, q)
			if err != nil {
				t.Fatal(err)
			}
			g := advice.Groups[0]
			if len(g.Results) != 0 || len(g.Rivals) != 3 || len(g.Coverage) != 1 || g.Coverage[0].Event != "50L" || g.Coverage[0].Swimmers != 3 || advice.RecommendedCategory != 0 {
				t.Fatalf("wrong coverage: %+v", g)
			}
			for i, r := range g.Rivals {
				if r.Mark.Centiseconds != 11500 || r.Club.ID != fmt.Sprintf("00%d", i+2) || r.Mark.Date != "2026-01-01" {
					t.Fatal("wrong best rival", r)
				}
			}
			d.Marks[3].Chrono = "E"
			advice, err = RecommendRelayCategories(context.Background(), d, q)
			if err != nil {
				t.Fatal(err)
			}
			g = advice.Groups[0]
			if len(g.Results) != 1 || g.Results[0].Rivals != len(g.Rivals) || g.Results[0].Rank != 4 || *g.Results[0].GapToPodium != 500 || g.Coverage[0].Swimmers != 4 {
				t.Fatalf("inconsistent comparison: %+v", g)
			}
		})
	}
}

func TestRelayCoverageDoesNotCountUnavailableOrWrongAgeAndKeepsStyles(t *testing.T) {
	d := adviceSquad()
	q := adviceRequest()
	q.Event = "4x50S"
	q.Category = 120
	d.Swimmers[0].Available = false
	d.Swimmers[1].BirthYear = 2004
	advice, err := RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	coverage := advice.Groups[0].Coverage
	if len(coverage) != 4 {
		t.Fatal(coverage)
	}
	for i, event := range []string{"50E", "50B", "50M", "50L"} {
		want := 0
		if i == 3 {
			want = 2
		}
		if coverage[i].Event != event || coverage[i].Swimmers != want {
			t.Fatal(coverage)
		}
	}
}

func TestRelayCombinesRawManualAndElectronicReferences(t *testing.T) {
	for _, gender := range []string{"M", "F", "X"} {
		t.Run(gender, func(t *testing.T) {
			d := adviceSquad()
			q := adviceRequest()
			q.Gender = gender
			q.Category = 120
			q.Chrono = ""
			for i := range d.Swimmers {
				g := gender
				if gender == "X" {
					g = "M"
					if i >= 2 {
						g = "F"
					}
				}
				d.Swimmers[i].Gender = g
				d.Marks[i].Gender = g
				if i%2 == 1 {
					d.Marks[i].Chrono = "M"
				}
				slower := d.Marks[i]
				slower.Chrono = "E"
				slower.Centiseconds = 3500
				d.Marks = append(d.Marks, slower)
			}
			for _, x := range []struct {
				id, chrono string
				cs         int
			}{{"002", "E", 12500}, {"002", "M", 11500}, {"003", "E", 13000}, {"004", "M", 14000}} {
				m := mark("", "4x50L", x.cs)
				m.ClubID = x.id
				m.Gender = gender
				m.Chrono = x.chrono
				m.RelayCategory = 120
				d.Marks = append(d.Marks, m)
			}
			otherPool := mark("", "4x50L", 9000)
			otherPool.ClubID = "005"
			otherPool.Gender = gender
			otherPool.Pool = 50
			otherPool.RelayCategory = 120
			d.Marks = append(d.Marks, otherPool)
			advice, err := RecommendRelayCategories(context.Background(), d, q)
			if err != nil {
				t.Fatal(err)
			}
			g := advice.Groups[0]
			if !g.AgeEligible || len(g.Results) != 1 || g.Results[0].Total != 12000 || g.Results[0].Rank != 2 || len(g.Rivals) != 3 || g.Rivals[0].Mark.Chrono != "M" || g.Rivals[0].Mark.Centiseconds != 11500 || *g.Results[0].GapToPodium != -2000 {
				t.Fatalf("wrong combined estimate: %+v", g)
			}
			e, m := 0, 0
			for _, l := range g.Results[0].Legs {
				if l.Chrono == "E" {
					e++
				} else if l.Chrono == "M" {
					m++
				}
				if l.Time != 3000 || l.Date != "2026-01-01" {
					t.Fatal(l)
				}
			}
			if e != 2 || m != 2 {
				t.Fatal("lost reference timings", g.Results[0].Legs)
			}
		})
	}
}

func TestRelayAgeEligibilityIndependentOfMarks(t *testing.T) {
	d := adviceSquad()
	d.Marks = nil
	q := adviceRequest()
	q.Chrono = ""
	for i := range d.Swimmers {
		d.Swimmers[i].BirthYear = 1997
	} // 4 x 30 = 120 in 2026/27.
	advice, err := RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range advice.Groups {
		if g.AgeEligible != (g.Category <= 120) || len(g.Results) != 0 {
			t.Fatal(g)
		}
	}
	d.Swimmers[0].BirthYear = 2004
	advice, err = RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range advice.Groups {
		if g.AgeEligible != (g.Category == 80) {
			t.Fatal("premaster eligibility", g)
		}
	}
	d.Swimmers[0].Available = false
	advice, err = RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range advice.Groups {
		if g.AgeEligible {
			t.Fatal("only three available", g)
		}
	}
	d.Swimmers[0].Available = true
	d.Swimmers[0].BirthYear = 1997
	q.Gender = "X"
	for i := range d.Swimmers {
		d.Swimmers[i].Gender = "F"
	}
	d.Swimmers[0].Gender = "M"
	advice, err = RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range advice.Groups {
		if g.AgeEligible {
			t.Fatal("mixed needs two men", g)
		}
	}
	d.Swimmers[1].Gender = "M"
	advice, err = RecommendRelayCategories(context.Background(), d, q)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range advice.Groups {
		if g.AgeEligible != (g.Category <= 120) {
			t.Fatal("mixed quartet eligibility", g)
		}
	}
}
