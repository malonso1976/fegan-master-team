package domain

import (
	"context"
	"errors"
)

// Rule identifiers describe competition types, independent of yearly circular URLs.
const ChampionshipRules = "championship"
const LeagueRules = "league"

// The accreditation requirement started on this date; it is not a rolling season cutoff.
const ChampionshipMarksSince = "2023-09-01"

type CategoryAdvice struct {
	Category    int           `json:"category"`
	Recommended bool          `json:"recommended"`
	Results     []RelayResult `json:"results"`
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
	if e := q.Filter.Validate(); e != nil {
		return out, e
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
		out.Groups = append(out.Groups, CategoryAdvice{Category: category, Results: results})
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
		out.Warnings = append(out.Warnings, "Sin recomendación competitiva de categoría: faltan al menos tres clubes rivales comparables para una referencia de podio.")
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
