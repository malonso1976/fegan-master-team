package domain

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
)

func TestTimeExact(t *testing.T) {
	for text, want := range map[string]int{"26.02": 2602, "1:02.34": 6234, "18:29.35": 110935, "0,5": 50, "60": 6000} {
		got, e := ParseTime(text)
		if e != nil || got != want {
			t.Fatalf("%s: %d %v", text, got, e)
		}
		again, e := ParseTime(FormatTime(got))
		if e != nil || again != got {
			t.Fatal("roundtrip")
		}
	}
	for _, text := range []string{"", "1:60.00", "-1", "1.234", "NaN", "0"} {
		if _, e := ParseTime(text); e == nil {
			t.Errorf("accepted %q", text)
		}
	}
}
func testFilter() Filter {
	return Filter{From: "2025-01-01", To: "2026-12-31", SeasonStart: 2025, Pool: 25, Chrono: "E", VerifiedOnly: true}
}
func mark(l, event string, cs int) Mark {
	return Mark{License: l, ClubID: "001", Event: event, Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: cs, Source: "manual", Verified: true, Gender: "M"}
}
func TestRecommendComparableDistinctRivals(t *testing.T) {
	d := Data{Swimmers: []Swimmer{{License: "a", BirthYear: 1986, Gender: "M"}, {License: "b", BirthYear: 1982, Gender: "M"}, {License: "c", BirthYear: 1981, Gender: "M"}, {License: "d", BirthYear: 1986, Gender: "F"}}}
	d.Marks = []Mark{mark("a", "50L", 3000), mark("b", "50L", 2900), mark("b", "50L", 2800), mark("c", "50L", 2000), mark("d", "50L", 1900)}
	otherPool := mark("b", "50L", 1000)
	otherPool.Pool = 50
	d.Marks = append(d.Marks, otherPool)
	unverified := mark("b", "50L", 900)
	unverified.Verified = false
	d.Marks = append(d.Marks, unverified)
	got, e := Recommend(d, "a", testFilter())
	if e != nil || len(got) != 1 {
		t.Fatal(got, e)
	}
	r := got[0]
	if r.Rank != 2 || r.Rivals != 1 || r.GapToFirst != 200 || r.AgeGroup != "40–44" {
		t.Fatalf("unexpected %+v", r)
	}
}
func TestRelayMatchesBruteForce(t *testing.T) {
	// Independent exhaustive oracle for small random squads, both race types and mixed categories.
	rng := rand.New(rand.NewSource(19))
	for run := 0; run < 20; run++ {
		d := Data{}
		times := map[string]map[string]int{}
		for i := 0; i < 7; i++ {
			l := fmt.Sprintf("s%d", i)
			gender := "M"
			if i%2 == 0 {
				gender = "F"
			}
			d.Swimmers = append(d.Swimmers, Swimmer{License: l, Name: l, BirthYear: 1980 + i*2, Gender: gender, ClubID: "001", Member: true, Available: true})
			times[l] = map[string]int{}
			for _, style := range []string{"L", "E", "B", "M"} {
				if rng.Intn(5) == 0 {
					continue
				}
				cs := 2200 + rng.Intn(1800)
				times[l][style] = cs
				m := mark(l, "50"+style, cs)
				m.Gender = gender
				d.Marks = append(d.Marks, m)
			}
		}
		for _, event := range []string{"4x50L", "4x50S"} {
			q := RelayRequest{Filter: testFilter(), ClubID: "001", Event: event, Gender: "X", MinAge: 20, MinTotal: 140, MaxTotal: 180, Limit: 1}
			got, e := Optimize(context.Background(), d, q)
			if e != nil {
				t.Fatal(e)
			}
			best := 1 << 30
			for a := 0; a < 7; a++ {
				for b := 0; b < 7; b++ {
					for c := 0; c < 7; c++ {
						for z := 0; z < 7; z++ {
							ids := []int{a, b, c, z}
							seen := map[int]bool{}
							age, fem, total := 0, 0, 0
							valid := true
							styles := []string{"L", "L", "L", "L"}
							if event == "4x50S" {
								styles = []string{"E", "B", "M", "L"}
							}
							for j, i := range ids {
								if seen[i] {
									valid = false
								}
								seen[i] = true
								s := d.Swimmers[i]
								age += SeasonAge(q.SeasonStart, s.BirthYear)
								if s.Gender == "F" {
									fem++
								}
								cs, ok := times[s.License][styles[j]]
								if !ok {
									valid = false
								}
								total += cs
							}
							if valid && fem == 2 && age >= q.MinTotal && age <= q.MaxTotal && total < best {
								best = total
							}
						}
					}
				}
			}
			if best == 1<<30 {
				if len(got) != 0 {
					t.Fatal("unexpected relay")
				}
			} else if len(got) != 1 || got[0].Total != best {
				t.Fatalf("%s run %d want %d got %+v", event, run, best, got)
			}
		}
	}
}
func TestRelayEligibilityAndRivalMarks(t *testing.T) {
	d := Data{}
	for i := 0; i < 4; i++ {
		l := fmt.Sprintf("s%d", i)
		d.Swimmers = append(d.Swimmers, Swimmer{License: l, BirthYear: 1996, Gender: "M", ClubID: "001", Member: true, Available: true})
		d.Marks = append(d.Marks, mark(l, "50L", 3000), mark(l, "50M", 1000))
	}
	rival := mark("", "4x50L", 11000)
	rival.ClubID = "002"
	rival.AgeTotal = 120
	rival.RelayCategory = 120
	d.Marks = append(d.Marks, rival)
	q := RelayRequest{Filter: testFilter(), ClubID: "001", Event: "4x50L", Category: 120, Gender: "M", MinAge: 20, MinTotal: 120, MaxTotal: 159, Limit: 5}
	got, e := Optimize(context.Background(), d, q)
	if e != nil || len(got) != 1 || got[0].Total != 12000 || got[0].Rank != 2 || got[0].Rivals != 1 {
		t.Fatal(got, e)
	}
	d.Swimmers[0].Available = false
	got, e = Optimize(context.Background(), d, q)
	if e != nil || len(got) != 0 {
		t.Fatal("unavailable swimmer used")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = Optimize(ctx, d, q); e == nil {
		t.Fatal("cancellation ignored")
	}
}
