package domain

import (
	"context"
	"errors"
)

const WinterRules = "https://www.fegan.org/fegan/tprd/2627/xeral/circulares/cir_2632_cto_gal_master_inv.pdf"
const LeagueRules = "https://www.fegan.org/fegan/tprd/2627/xeral/circulares/cir_2630_liga_gal_master.pdf"

type CategoryAdvice struct {
	Category    int           `json:"category"`
	Recommended bool          `json:"recommended"`
	Results     []RelayResult `json:"results"`
}
type RelayAdvice struct {
	Season              string           `json:"season"`
	AgeAt               string           `json:"age_at"`
	Source              string           `json:"source"`
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
		out.Source = WinterRules
		out.Warnings = append(out.Warnings, "Criterios comunes de edad de las circulares 26-32 y 26-30. Revisa acreditación y límites de participación antes de inscribir.")
	case "winter2627":
		out.Source = WinterRules
		if q.From < "2023-09-01" {
			q.From = "2023-09-01"
		}
		q.VerifiedOnly = true
		out.Warnings = append(out.Warnings, "Invierno: solo referencias verificadas desde 01/09/2023; revisa su acreditación oficial antes de inscribir.")
	case "league2627":
		out.Source = LeagueRules
		out.Warnings = append(out.Warnings, "Liga: una remuda por nadador y jornada; estas alternativas no forman una planificación conjunta.")
	default:
		return out, errors.New("selecciona normativa de invierno o liga 2026/27")
	}
	if q.SeasonStart != 2026 {
		out.Warnings = append(out.Warnings, "Simulación: se aplican reglas publicadas para 2026/27 a otra temporada; comprueba la circular correspondiente.")
	}
	if q.Event != "4x50L" && q.Event != "4x50S" {
		return out, errors.New("las circulares seleccionadas contemplan relevos 4x50 libre y estilos")
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
