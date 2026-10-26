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
}

type Recommendation struct {
	Details    []RivalDetail `json:"details"`
	OwnMark    Mark          `json:"own_mark"`
	Pool       int           `json:"pool"`
	Chrono     string        `json:"chrono"`
	Event      string        `json:"event"`
	Time       int           `json:"time"`
	Rank       int           `json:"rank"`
	Rivals     int           `json:"rivals"`
	GapToFirst int           `json:"gap_to_first"`
	AgeGroup   string        `json:"age_group"`
}

func Recommend(d Data, license string, f Filter) ([]Recommendation, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	var target Swimmer
	found := false
	for _, s := range d.Swimmers {
		if s.License == license {
			target = s
			found = true
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
	eligible := map[string]bool{}
	for _, s := range d.Swimmers {
		a := SeasonAge(f.SeasonStart, s.BirthYear)
		if s.Gender == target.Gender && a >= low && (high == 0 || a <= high) {
			eligible[s.License] = true
		}
	}
	best := map[string]map[string]int{}
	for _, m := range d.Marks {
		if Relay(m.Event) || !eligible[m.License] || m.Gender != target.Gender || !f.accepts(m) {
			continue
		}
		if best[m.Event] == nil {
			best[m.Event] = map[string]int{}
		}
		v, ok := best[m.Event][m.License]
		if !ok || m.Centiseconds < v {
			best[m.Event][m.License] = m.Centiseconds
		}
	}
	out := []Recommendation{}
	for event, times := range best {
		own, ok := times[license]
		if !ok {
			continue
		}
		rank, first := 1, own
		type rival struct {
			license string
			time    int
		}
		observed := []rival{}
		for l, t := range times {
			observed = append(observed, rival{l, t})
		}
		sort.Slice(observed, func(i, j int) bool {
			if observed[i].time != observed[j].time {
				return observed[i].time < observed[j].time
			}
			return observed[i].license < observed[j].license
		})
		if len(observed) > 10 {
			observed = observed[:10]
		}
		count := 0
		for _, r := range observed {
			if r.license != license {
				count++
				if r.time < own {
					rank++
				}
			}
			if r.time < first {
				first = r.time
			}
		}
		details := []RivalDetail{}
		bestMark := func(l string) Mark {
			var chosen Mark
			for _, m := range d.Marks {
				if m.License == l && m.Event == event && f.accepts(m) && m.Gender == target.Gender && (chosen.Date == "" || m.Centiseconds < chosen.Centiseconds || (m.Centiseconds == chosen.Centiseconds && m.Date > chosen.Date)) {
					chosen = m
				}
			}
			return chosen
		}
		for _, r := range observed {
			if r.license == license {
				continue
			}
			for _, p := range d.Swimmers {
				if p.License == r.license {
					details = append(details, RivalDetail{p, bestMark(r.license)})
					break
				}
			}
		}
		out = append(out, Recommendation{Event: event, Time: own, Rank: rank, Rivals: count, GapToFirst: own - first, AgeGroup: groupLabel(low, high), Pool: f.Pool, Chrono: f.Chrono, Details: details, OwnMark: bestMark(license)})
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

func Optimize(ctx context.Context, d Data, q RelayRequest) ([]RelayResult, error) {
	if err := q.Filter.Validate(); err != nil {
		return nil, err
	}
	if !Relay(q.Event) || !ValidEvent(q.Event) || !ValidID(q.ClubID) || (q.Gender != "M" && q.Gender != "F" && q.Gender != "X") || q.MinTotal < 0 || q.MaxTotal < q.MinTotal || q.MaxTotal > 500 || q.MinAge < 18 || q.Limit < 1 || q.Limit > 20 {
		return nil, errors.New("configuración de relevo no válida")
	}
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
	best := map[string]map[string]int{}
	for _, m := range d.Marks {
		s, ok := swimmers[m.License]
		if !ok || m.Gender != s.Gender || !q.Filter.accepts(m) || Relay(m.Event) {
			continue
		}
		if best[m.License] == nil {
			best[m.License] = map[string]int{}
		}
		old, ok := best[m.License][m.Event]
		if !ok || m.Centiseconds < old {
			best[m.License][m.Event] = m.Centiseconds
		}
	}
	var candidates [4][]Leg
	for i, style := range styles {
		event := distance + style
		for l, s := range swimmers {
			if t, ok := best[l][event]; ok {
				candidates[i] = append(candidates[i], Leg{l, s.Name, event, t, SeasonAge(q.SeasonStart, s.BirthYear)})
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
			if swimmers[leg.License].Gender == "F" {
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
	rivals := map[string]int{}
	for _, m := range d.Marks {
		if m.ClubID != q.ClubID && m.Event == q.Event && m.Gender == q.Gender && m.RelayCategory == q.Category && q.Filter.accepts(m) {
			old, ok := rivals[m.ClubID]
			if !ok || m.Centiseconds < old {
				rivals[m.ClubID] = m.Centiseconds
			}
		}
	}
	rivalTimes := []int{}
	for _, v := range rivals {
		rivalTimes = append(rivalTimes, v)
	}
	sort.Ints(rivalTimes)
	for i := range out {
		if len(rivalTimes) >= 3 {
			gap := out[i].Total - rivalTimes[2]
			out[i].GapToPodium = &gap
		}
		out[i].Rivals = len(rivals)
		out[i].Rank = 1
		for _, t := range rivals {
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
