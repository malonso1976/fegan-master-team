package fegan

import (
	"context"
	"errors"
	"fmt"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"time"
)

// ExpandQuery enumerates actual FEGAN filters, without weighted conversions.
func ExpandQuery(q Query) ([]Query, error) {
	pools := []int{q.Pool}
	if q.Pool == 0 {
		pools = []int{25, 50}
	}
	chronos := []string{q.Chrono}
	if q.Chrono == "" {
		chronos = []string{"E", "M"}
	}
	genders := []string{q.Gender}
	if q.Gender == "" {
		genders = []string{"M", "F"}
	}
	out := []Query{}
	for _, pool := range pools {
		for _, chrono := range chronos {
			for _, gender := range genders {
				v := q
				v.Pool = pool
				v.Chrono = chrono
				v.Gender = gender
				check := v
				if check.Event == "all" {
					check.Event = "50L"
				}
				if e := check.Validate(); e != nil {
					return nil, e
				}
				out = append(out, v)
			}
		}
	}
	return out, nil
}
func (c *Client) WidePreview(ctx context.Context, q Query) (Preview, error) {
	queries, e := ExpandQuery(q)
	if e != nil {
		return Preview{}, e
	}
	out := Preview{Query: q, Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}, Cached: true, Warnings: []string{"Incluye las piscinas y cronometrajes consultados por separado, sin equivalencias."}}
	clubs := map[string]domain.Club{}
	people := map[string]domain.Swimmer{}
	for _, v := range queries {
		p, e := c.Preview(ctx, v)
		if e != nil {
			return Preview{}, e
		}
		out.Requests += p.Requests
		out.Cached = out.Cached && p.Cached
		out.Data.Marks = append(out.Data.Marks, p.Data.Marks...)
		for _, c := range p.Data.Clubs {
			clubs[c.ID] = c
		}
		for _, s := range p.Data.Swimmers {
			if old, ok := people[s.License]; ok && (old.BirthYear != s.BirthYear || old.Gender != s.Gender) {
				return Preview{}, errors.New("conflicto de identidad entre consultas")
			}
			people[s.License] = s
		}
		out.Warnings = append(out.Warnings, p.Warnings...)
	}
	for _, c := range clubs {
		out.Data.Clubs = append(out.Data.Clubs, c)
	}
	for _, s := range people {
		out.Data.Swimmers = append(out.Data.Swimmers, s)
	}
	if q.License != "" && q.Gender != "" {
		personal, err := c.PersonalPreview(ctx, q)
		if err != nil {
			out.Warnings = append(out.Warnings, "No se pudo completar la consulta personal FEGAN; la vista previa solo contiene el historial general validado.")
		} else if err = mergePersonal(&out, personal); err != nil {
			return Preview{}, err
		}
	}
	out.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	return out, nil
}

func mergePersonal(out *Preview, personal Preview) error {
	people := map[string]domain.Swimmer{}
	clubs := map[string]bool{}
	marks := map[string]bool{}
	key := func(m domain.Mark) string {
		return fmt.Sprintf("%q/%q/%q/%d/%q/%q/%d/%q/%q", m.License, m.ClubID, m.Event, m.Pool, m.Chrono, m.Date, m.Centiseconds, m.Location, m.Gender)
	}
	for _, s := range out.Data.Swimmers {
		people[s.License] = s
	}
	for _, c := range out.Data.Clubs {
		clubs[c.ID] = true
	}
	for _, m := range out.Data.Marks {
		marks[key(m)] = true
	}
	for _, s := range personal.Data.Swimmers {
		if old, ok := people[s.License]; ok {
			if old.BirthYear != s.BirthYear || old.Gender != s.Gender {
				return errors.New("identidad personal incompatible con el historial general")
			}
		} else {
			out.Data.Swimmers = append(out.Data.Swimmers, s)
			people[s.License] = s
		}
	}
	for _, c := range personal.Data.Clubs {
		if !clubs[c.ID] {
			out.Data.Clubs = append(out.Data.Clubs, c)
			clubs[c.ID] = true
		}
	}
	for _, m := range personal.Data.Marks {
		if !marks[key(m)] {
			out.Data.Marks = append(out.Data.Marks, m)
			marks[key(m)] = true
		}
	}
	out.Requests += personal.Requests
	out.Cached = out.Cached && personal.Cached
	out.Warnings = append(out.Warnings, personal.Warnings...)
	return nil
}
