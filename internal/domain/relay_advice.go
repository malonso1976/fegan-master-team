package domain

import (
	"context"
	"errors"
	"sort"
)

// Rule identifiers describe competition types, independent of yearly circular URLs.
const ChampionshipRules = "championship"
const LeagueRules = "league"

// The accreditation requirement started on this date; it is not a rolling season cutoff.
const ChampionshipMarksSince = "2023-09-01"

type RelayRival struct {
	Club Club `json:"club"`
	Mark Mark `json:"mark"`
}
type RelayReferenceCoverage struct {
	Event    string `json:"event"`
	Swimmers int    `json:"swimmers"`
}

type CategoryAdvice struct {
	AgeEligible bool                     `json:"age_eligible"`
	Rivals      []RelayRival             `json:"rivals"`
	Coverage    []RelayReferenceCoverage `json:"coverage"`
	Category    int                      `json:"category"`
	Recommended bool                     `json:"recommended"`
	Results     []RelayResult            `json:"results"`
}
type RelayAdvice struct {
	Season              string           `json:"season"`
	AgeAt               string           `json:"age_at"`
	Rules               string           `json:"rules"`
	RecommendedCategory int              `json:"recommended_category"`
	Groups              []CategoryAdvice `json:"groups"`
	Warnings            []string         `json:"warnings"`
}

// Category eligibility follows the user's clarification of participation section 3.3:
// non-premasters may enter lower categories; premaster swimmers may only enter +80.
func RecommendRelayCategories(ctx context.Context, d Data, q RelayRequest) (RelayAdvice, error) {
	out := RelayAdvice{Season: SeasonLabel(q.SeasonStart), AgeAt: SeasonLabel(q.SeasonStart), Groups: []CategoryAdvice{}, Warnings: []string{"Propuestas alternativas e independientes. No son inscripciones ni garantizan medallas.", "La posición usa la mejor marca por club rival y categoría guardada; la cobertura puede ser incompleta."}}
	if e := validateRelayFilter(q.Filter); e != nil {
		return out, e
	}
	if q.Chrono == "" {
		out.Warnings = append(out.Warnings, "Tiempos manuales y electrónicos usados tal como se registraron, sin conversión.")
	}
	if q.Category != 0 && !ValidRelayCategory(q.Category) {
		return out, errors.New("categoría de relevos no válida")
	}
	switch q.Competition {
	case "":
		out.Warnings = append(out.Warnings, "Criterios comunes de edad. Revisa acreditación y límites de participación de la competición antes de inscribir.")
	case ChampionshipRules:
		out.Rules = ChampionshipRules
		if q.From < ChampionshipMarksSince {
			q.From = ChampionshipMarksSince
		}
		q.VerifiedOnly = true
		out.Warnings = append(out.Warnings, "Campeonato: requiere marca acreditada desde 01/09/2023, fecha histórica de entrada en vigor de la exigencia. La verificación local no acredita una inscripción federativa.")
	case LeagueRules:
		out.Rules = LeagueRules
		out.Warnings = append(out.Warnings, "Liga: una remuda por nadador y jornada; estas alternativas no forman una planificación conjunta.")
	default:
		return out, errors.New("tipo de competición no válido: championship o league")
	}
	if q.Event != "4x50L" && q.Event != "4x50S" {
		return out, errors.New("el programa de relevos implementado contempla relevos 4x50 libre y estilos")
	}
	out.AgeAt = ageReference(q.SeasonStart)
	for _, category := range RelayCategories() {
		if q.Category != 0 && q.Category != category {
			continue
		}
		candidate := q
		candidate.Category = category
		candidate.MinTotal = category
		candidate.MaxTotal = 500
		candidate.MinAge = 25
		if category == 80 {
			candidate.MinAge = 20
		}
		results, e := Optimize(ctx, d, candidate)
		if e != nil {
			return out, e
		}
		candidates, styles := relayCandidates(d, candidate)
		coverage := []RelayReferenceCoverage{}
		seen := map[string]bool{}
		for i, style := range styles {
			event := "50" + style
			if !seen[event] {
				coverage = append(coverage, RelayReferenceCoverage{Event: event, Swimmers: len(candidates[i])})
				seen[event] = true
			}
		}
		out.Groups = append(out.Groups, CategoryAdvice{AgeEligible: relayAgeEligible(d, candidate), Category: category, Results: results, Rivals: comparableRelayRivals(d, candidate), Coverage: coverage})
	}
	best := -1
	for i, g := range out.Groups {
		if len(g.Results) == 0 || g.Results[0].Rivals < 3 {
			continue
		}
		if best < 0 || betterOpportunity(g.Results[0], out.Groups[best].Results[0]) {
			best = i
		}
	}
	if best >= 0 {
		out.Groups[best].Recommended = true
		out.RecommendedCategory = out.Groups[best].Category
	} else {
		out.Warnings = append(out.Warnings, "Sin recomendación competitiva de categoría: se necesita una composición propia y al menos tres clubes rivales comparables en el mismo grupo.")
	}
	return out, nil
}
func betterOpportunity(a, b RelayResult) bool {
	if a.Rank != b.Rank {
		return a.Rank < b.Rank
	}
	if a.GapToPodium != nil && b.GapToPodium != nil {
		aThird := a.Total - *a.GapToPodium
		bThird := b.Total - *b.GapToPodium
		return int64(*a.GapToPodium)*int64(bThird) < int64(*b.GapToPodium)*int64(aThird)
	}
	return a.Total < b.Total
}

// Keep the actual best result per rival club, even without a feasible own squad.
func comparableRelayRivals(d Data, q RelayRequest) []RelayRival {
	best := map[string]Mark{}
	clubs := map[string]Club{}
	for _, club := range d.Clubs {
		clubs[club.ID] = club
	}
	for _, m := range d.Marks {
		if m.ClubID == q.ClubID || m.Event != q.Event || m.Gender != q.Gender || m.RelayCategory != q.Category || !relayAccepts(q.Filter, m) {
			continue
		}
		old, ok := best[m.ClubID]
		if !ok || betterReference(m, old) {
			best[m.ClubID] = m
		}
	}
	out := make([]RelayRival, 0, len(best))
	for id, mark := range best {
		club, ok := clubs[id]
		if !ok {
			club = Club{ID: id}
		}
		out = append(out, RelayRival{Club: club, Mark: mark})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mark.Centiseconds != out[j].Mark.Centiseconds {
			return out[i].Mark.Centiseconds < out[j].Mark.Centiseconds
		}
		return out[i].Mark.ClubID < out[j].Mark.ClubID
	})
	return out
}

// Empty timing means the raw E/M times may be combined for relay estimates.
func validateRelayFilter(f Filter) error {
	if f.Chrono == "" {
		f.Chrono = "E"
	}
	return f.Validate()
}
func relayAccepts(f Filter, m Mark) bool {
	if m.Chrono != "E" && m.Chrono != "M" {
		return false
	}
	if f.Chrono == "" {
		f.Chrono = m.Chrono
	}
	return f.accepts(m)
}

// Age eligibility is independent of times. The small subset-sum state avoids
// enumerating all quartets and preserves the upper bound and mixed composition.
func relayAgeEligible(d Data, q RelayRequest) bool {
	var states [5]map[int]bool
	for i := range states {
		states[i] = map[int]bool{}
	}
	states[0][0] = true
	for _, s := range d.Swimmers {
		age := SeasonAge(q.SeasonStart, s.BirthYear)
		if s.ClubID != q.ClubID || !s.Member || !s.Available || age < q.MinAge || (q.Gender != "X" && s.Gender != q.Gender) {
			continue
		}
		female := 0
		if s.Gender == "F" {
			female = 1
		}
		for n := 4; n >= 1; n-- {
			for key := range states[n-1] {
				sum, females := key/5+age, key%5+female
				if sum > q.MaxTotal || (q.Gender == "X" && females > 2) {
					continue
				}
				if n == 4 && sum >= q.MinTotal && (q.Gender != "X" || females == 2) {
					return true
				}
				states[n][sum*5+females] = true
			}
		}
	}
	return false
}
