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
