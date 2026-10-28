package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Filter struct {
	From         string `json:"from"`
	To           string `json:"to"`
	SeasonStart  int    `json:"season_start"`
	Pool         int    `json:"pool"`
	Chrono       string `json:"chrono"`
	VerifiedOnly bool   `json:"verified_only"`
}

func (f Filter) Validate() error {
	a, e := time.Parse("2006-01-02", f.From)
	if e != nil {
		return errors.New("fecha desde no válida")
	}
	b, e := time.Parse("2006-01-02", f.To)
	if e != nil || a.After(b) || f.SeasonStart < 1900 || f.SeasonStart > 2199 || (f.Pool != 25 && f.Pool != 50) || (f.Chrono != "M" && f.Chrono != "E") {
		return errors.New("revisar fechas, temporada, piscina y cronometraje")
	}
	return nil
}
func (f Filter) accepts(m Mark) bool {
	return m.Date >= f.From && m.Date <= f.To && m.Pool == f.Pool && m.Chrono == f.Chrono && (!f.VerifiedOnly || m.Verified)
}

type RivalDetail struct {
	Swimmer Swimmer `json:"swimmer"`
	Mark    Mark    `json:"mark"`
	Time    int     `json:"time"`
}

type Recommendation struct {
	Conversion *ConversionInfo `json:"conversion,omitempty"`
	Warnings   []string        `json:"warnings,omitempty"`
	Details    []RivalDetail   `json:"details"`
	OwnMark    Mark            `json:"own_mark"`
	Pool       int             `json:"pool"`
	Chrono     string          `json:"chrono"`
	Event      string          `json:"event"`
	Time       int             `json:"time"`
	Rank       int             `json:"rank"`
	Rivals     int             `json:"rivals"`
	GapToFirst int             `json:"gap_to_first"`
	AgeGroup   string          `json:"age_group"`
}

// betterReference breaks ties deterministically without converting timings.
func betterReference(a, b Mark) bool {
	if a.Centiseconds != b.Centiseconds {
		return a.Centiseconds < b.Centiseconds
	}
	if a.Chrono != b.Chrono {
		return a.Chrono == "E"
	}
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	return a.ID < b.ID
}

// Recommend uses the swimmer's lifetime best up to To, and rivals only in
// From..To. Each event uses the timing type of its own historical best.
func Recommend(d Data, license string, f Filter) ([]Recommendation, error) {
	if f.Pool == 0 {
		return recommendConverted(d, license, f)
	}
	check := f
	if check.Chrono == "" {
		check.Chrono = "E"
	}
	if e := check.Validate(); e != nil {
		return nil, e
	}
	var target Swimmer
	found := false
	for _, s := range d.Swimmers {
		if s.License == license {
			target = s
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("nadador no encontrado")
	}
	age := SeasonAge(f.SeasonStart, target.BirthYear)
	if age < 20 {
		return nil, errors.New("nadador por debajo de la edad mínima del escenario")
	}
	low, high := IndividualGroup(age)
	eligible := map[string]Swimmer{}
	for _, s := range d.Swimmers {
		a := SeasonAge(f.SeasonStart, s.BirthYear)
		if s.License != license && s.Gender == target.Gender && a >= low && (high == 0 || a <= high) {
			eligible[s.License] = s
		}
	}
	ownBest := map[string]Mark{}
	for _, m := range d.Marks {
		if m.License != license || Relay(m.Event) || m.Gender != target.Gender || m.Pool != f.Pool || m.Date > f.To || m.Centiseconds <= 0 || (f.Chrono != "" && m.Chrono != f.Chrono) || (m.Chrono != "E" && m.Chrono != "M") || (f.VerifiedOnly && !m.Verified) {
			continue
		}
		old, ok := ownBest[m.Event]
		if !ok || betterReference(m, old) {
			ownBest[m.Event] = m
		}
	}
	out := []Recommendation{}
	for event, own := range ownBest {
		comparable := f
		comparable.Chrono = own.Chrono
		best := map[string]Mark{}
		for _, m := range d.Marks {
			if _, ok := eligible[m.License]; !ok || m.Event != event || m.Gender != target.Gender || m.Centiseconds <= 0 || !comparable.accepts(m) {
				continue
			}
			old, ok := best[m.License]
			if !ok || betterReference(m, old) {
				best[m.License] = m
			}
		}
		observed := []Mark{}
		for _, m := range best {
			observed = append(observed, m)
		}
		sort.Slice(observed, func(i, j int) bool {
			if observed[i].Centiseconds != observed[j].Centiseconds {
				return observed[i].Centiseconds < observed[j].Centiseconds
			}
			return observed[i].License < observed[j].License
		})
		if len(observed) > 10 {
			observed = observed[:10]
		}
		rank, first := 1, own.Centiseconds
		details := []RivalDetail{}
		for _, m := range observed {
			if m.Centiseconds < own.Centiseconds {
				rank++
			}
			if m.Centiseconds < first {
				first = m.Centiseconds
			}
			details = append(details, RivalDetail{Swimmer: eligible[m.License], Mark: m, Time: m.Centiseconds})
		}
		out = append(out, Recommendation{Details: details, OwnMark: own, Pool: f.Pool, Chrono: own.Chrono, Event: event, Time: own.Centiseconds, Rank: rank, Rivals: len(observed), GapToFirst: own.Centiseconds - first, AgeGroup: groupLabel(low, high)})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.Rivals > 0) != (b.Rivals > 0) {
			return a.Rivals > 0
		}
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		if a.GapToFirst != b.GapToFirst {
			return a.GapToFirst < b.GapToFirst
		}
		return a.Event < b.Event
	})
	return out, nil
}

type RelayRequest struct {
	Filter
	Category    int    `json:"category"`
	Competition string `json:"competition"`
	ClubID      string `json:"club_id"`
	Event       string `json:"event"`
	Gender      string `json:"gender"`
	MinTotal    int    `json:"-"`
	MaxTotal    int    `json:"-"`
	MinAge      int    `json:"-"`
	Limit       int    `json:"limit"`
}
type Leg struct {
	Chrono  string `json:"chrono"`
	Date    string `json:"date"`
	License string `json:"license"`
	Name    string `json:"name"`
	Event   string `json:"event"`
	Time    int    `json:"time"`
	Age     int    `json:"age"`
}
type RelayResult struct {
	Legs        []Leg `json:"legs"`
	Total       int   `json:"total"`
	AgeTotal    int   `json:"age_total"`
	Rivals      int   `json:"rivals"`
	Rank        int   `json:"rank"`
	GapToPodium *int  `json:"gap_to_podium"`
}

func relayCandidates(d Data, q RelayRequest) ([4][]Leg, []string) {
	// The four events are in race order: backstroke, breaststroke, butterfly, freestyle.
	distance := strings.TrimSuffix(strings.TrimPrefix(q.Event, "4x"), "L")
	styles := []string{"L", "L", "L", "L"}
	if strings.HasSuffix(q.Event, "S") {
		distance = strings.TrimSuffix(strings.TrimPrefix(q.Event, "4x"), "S")
		styles = []string{"E", "B", "M", "L"}
	}
	swimmers := map[string]Swimmer{}
	for _, s := range d.Swimmers {
		if s.ClubID == q.ClubID && s.Member && s.Available && SeasonAge(q.SeasonStart, s.BirthYear) >= q.MinAge && (q.Gender == "X" || s.Gender == q.Gender) {
			swimmers[s.License] = s
		}
	}
	best := map[string]map[string]Mark{}
	for _, m := range d.Marks {
		s, ok := swimmers[m.License]
		if !ok || m.Gender != s.Gender || !relayAccepts(q.Filter, m) || Relay(m.Event) {
			continue
		}
		if best[m.License] == nil {
			best[m.License] = map[string]Mark{}
		}
		old, ok := best[m.License][m.Event]
		if !ok || betterReference(m, old) {
			best[m.License][m.Event] = m
		}
	}
	var candidates [4][]Leg
	for i, style := range styles {
		event := distance + style
		for l, s := range swimmers {
			if t, ok := best[l][event]; ok {
				candidates[i] = append(candidates[i], Leg{License: l, Name: s.Name, Event: event, Time: t.Centiseconds, Age: SeasonAge(q.SeasonStart, s.BirthYear), Chrono: t.Chrono, Date: t.Date})
			}
		}
		sort.Slice(candidates[i], func(a, b int) bool {
			x, y := candidates[i][a], candidates[i][b]
			if x.Time != y.Time {
				return x.Time < y.Time
			}
			return x.License < y.License
		})
	}
	return candidates, styles
}

func validateRelayRequest(q RelayRequest) error {
	if err := validateRelayFilter(q.Filter); err != nil {
		return err
	}
	if !Relay(q.Event) || !ValidEvent(q.Event) || !ValidID(q.ClubID) || (q.Gender != "M" && q.Gender != "F" && q.Gender != "X") || q.MinTotal < 0 || q.MaxTotal < q.MinTotal || q.MaxTotal > 500 || q.MinAge < 18 || q.Limit < 1 || q.Limit > 20 {
		return errors.New("configuración de relevo no válida")
	}
	return nil
}

func Optimize(ctx context.Context, d Data, q RelayRequest) ([]RelayResult, error) {
	if err := validateRelayRequest(q); err != nil {
		return nil, err
	}
	candidates, styles := relayCandidates(d, q)
	genders := map[string]string{}
	for _, swimmer := range d.Swimmers {
		genders[swimmer.License] = swimmer.Gender
	}
	out := []RelayResult{}
	used := map[string]bool{}
	legs := []Leg{}
	nodes := 0
	var search func(int, int, int, int) error
	search = func(depth, total, age, females int) error {
		nodes++
		if nodes > 2_000_000 {
			return errors.New("escenario demasiado grande; reduce nadadores disponibles o el intervalo de edades")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if age > q.MaxTotal {
			return nil
		}
		if q.Gender == "X" && (females > 2 || females+4-depth < 2) {
			return nil
		}
		if depth == 4 {
			if age < q.MinTotal || age > q.MaxTotal || (q.Gender == "X" && females != 2) {
				return nil
			}
			premaster := false
			for _, leg := range legs {
				premaster = premaster || leg.Age < 25
			}
			if !relayProposalAllowed(q.Category, age, premaster) {
				return nil
			}
			out = append(out, RelayResult{Legs: append([]Leg{}, legs...), Total: total, AgeTotal: age})
			sort.SliceStable(out, func(i, j int) bool { return out[i].Total < out[j].Total })
			if len(out) > q.Limit {
				out = out[:q.Limit]
			}
			return nil
		}
		lower := total
		for i := depth; i < 4; i++ {
			if len(candidates[i]) == 0 {
				return nil
			}
			lower += candidates[i][0].Time
		}
		if len(out) == q.Limit && lower > out[len(out)-1].Total {
			return nil
		}
		for _, leg := range candidates[depth] {
			if used[leg.License] {
				continue
			}
			if styles[0] == "L" && depth > 0 && leg.License <= legs[depth-1].License {
				continue
			}
			used[leg.License] = true
			legs = append(legs, leg)
			female := 0
			if genders[leg.License] == "F" {
				female = 1
			}
			if err := search(depth+1, total+leg.Time, age+leg.Age, females+female); err != nil {
				return err
			}
			legs = legs[:len(legs)-1]
			delete(used, leg.License)
		}
		return nil
	}
	if err := search(0, 0, 0, 0); err != nil {
		return nil, err
	}
	rivals := comparableRelayRivals(d, q)
	rivalTimes := make([]int, len(rivals))
	for i, rival := range rivals {
		rivalTimes[i] = rival.Mark.Centiseconds
	}
	for i := range out {
		if len(rivalTimes) >= 3 {
			gap := out[i].Total - rivalTimes[2]
			out[i].GapToPodium = &gap
		}
		out[i].Rivals = len(rivals)
		out[i].Rank = 1
		for _, t := range rivalTimes {
			if t < out[i].Total {
				out[i].Rank++
			}
		}
	}
	return out, nil
}

func groupLabel(low, high int) string {
	if high == 0 {
		return fmt.Sprintf("%d+", low)
	}
	return fmt.Sprintf("%d–%d", low, high)
}
