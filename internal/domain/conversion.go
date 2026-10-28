package domain

import (
	_ "embed"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// User-selected FMN table. Edition is provenance, not an annual eligibility rule.
//
//go:embed tables/conversion.json
var conversionJSON []byte

type ConversionInfo struct {
	Edition   string `json:"edition"`
	SourceURL string `json:"source_url"`
}
type conversionTable struct {
	ConversionInfo
	ManualToElectronic   int                       `json:"manual_to_electronic"`
	Manual50ToElectronic int                       `json:"manual_50_to_electronic"`
	PoolOffsets          map[string]map[string]int `json:"pool_offsets"`
}

var conversions = func() conversionTable {
	var table conversionTable
	if err := json.Unmarshal(conversionJSON, &table); err != nil {
		panic(err)
	}
	return table
}()

// ComparablePools never extrapolates blank cells in the supplied table.
func ComparablePools(event, gender string, original int) []int {
	if _, ok := conversions.PoolOffsets[gender][event]; ok {
		return []int{25, 50}
	}
	return []int{original}
}

// ConvertedTime expresses a reference in electronic timing without changing it.
// Pool adjustments are additive (25 -> 50) or subtractive (50 -> 25).
func ConvertedTime(m Mark, pool int) (int, bool) {
	if (m.Pool != 25 && m.Pool != 50) || (pool != 25 && pool != 50) || (m.Chrono != "E" && m.Chrono != "M") || m.Centiseconds <= 0 || m.Centiseconds > 24*60*60*100 {
		return 0, false
	}
	cs := m.Centiseconds
	if pool != m.Pool {
		offset, ok := conversions.PoolOffsets[m.Gender][m.Event]
		if !ok {
			return 0, false
		}
		if pool == 50 {
			cs += offset
		} else {
			cs -= offset
		}
	}
	if m.Chrono == "M" {
		correction := conversions.ManualToElectronic
		// Only individual 50 m events use 0.29; a 4x50 relay is not a 50 m event.
		if strings.HasPrefix(m.Event, "50") {
			correction = conversions.Manual50ToElectronic
		}
		cs += correction
	}
	return cs, cs > 0
}

type convertedReference struct {
	mark Mark
	time int
}

func (a convertedReference) better(b convertedReference) bool {
	if a.time != b.time {
		return a.time < b.time
	}
	if a.mark.Chrono != b.mark.Chrono {
		return a.mark.Chrono == "E"
	}
	if a.mark.Date != b.mark.Date {
		return a.mark.Date > b.mark.Date
	}
	if a.mark.ID != b.mark.ID {
		return a.mark.ID < b.mark.ID
	}
	return a.mark.Pool < b.mark.Pool
}

type comparisonKey struct {
	event string
	pool  int
}

func recommendConverted(d Data, license string, f Filter) ([]Recommendation, error) {
	check := f
	check.Pool = 25
	if check.Chrono == "" {
		check.Chrono = "E"
	}
	if err := check.Validate(); err != nil {
		return nil, err
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
	usable := func(m Mark) bool {
		return !Relay(m.Event) && m.Gender == target.Gender && m.Date <= f.To && (!f.VerifiedOnly || m.Verified) && (f.Chrono == "" || m.Chrono == f.Chrono)
	}
	ownBest := map[comparisonKey]convertedReference{}
	for _, m := range d.Marks {
		if m.License != license || !usable(m) {
			continue
		}
		pool := m.Pool
		if _, ok := conversions.PoolOffsets[m.Gender][m.Event]; ok {
			pool = 50
		}
		cs, ok := ConvertedTime(m, pool)
		if !ok {
			continue
		}
		key := comparisonKey{m.Event, pool}
		ref := convertedReference{m, cs}
		old, exists := ownBest[key]
		if !exists || ref.better(old) {
			ownBest[key] = ref
		}
	}
	out := []Recommendation{}
	for key, own := range ownBest {
		best := map[string]convertedReference{}
		for _, m := range d.Marks {
			if _, ok := eligible[m.License]; !ok || m.Event != key.event || !usable(m) || m.Date < f.From {
				continue
			}
			cs, ok := ConvertedTime(m, key.pool)
			if !ok {
				continue
			}
			ref := convertedReference{m, cs}
			old, exists := best[m.License]
			if !exists || ref.better(old) {
				best[m.License] = ref
			}
		}
		observed := []convertedReference{}
		for _, ref := range best {
			observed = append(observed, ref)
		}
		sort.Slice(observed, func(i, j int) bool {
			if observed[i].time != observed[j].time {
				return observed[i].time < observed[j].time
			}
			return observed[i].mark.License < observed[j].mark.License
		})
		if len(observed) > 10 {
			observed = observed[:10]
		}
		r := Recommendation{OwnMark: own.mark, Pool: key.pool, Chrono: "E", Event: key.event, Time: own.time, Rank: 1, Rivals: len(observed), AgeGroup: groupLabel(low, high), Conversion: &conversions.ConversionInfo, Details: []RivalDetail{}}
		first := own.time
		for _, ref := range observed {
			if ref.time < own.time {
				r.Rank++
			}
			if ref.time < first {
				first = ref.time
			}
			r.Details = append(r.Details, RivalDetail{Swimmer: eligible[ref.mark.License], Mark: ref.mark, Time: ref.time})
		}
		r.GapToFirst = own.time - first
		if _, ok := conversions.PoolOffsets[target.Gender][key.event]; !ok {
			r.Warnings = []string{"Sin ajuste de piscina en la tabla: se comparan solo marcas de esta piscina, con cronometraje convertido a electrónico."}
		}
		out = append(out, r)
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
		if a.Event != b.Event {
			return a.Event < b.Event
		}
		return a.Pool < b.Pool
	})
	return out, nil
}
