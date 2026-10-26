package fegan

import (
	"context"
	"errors"
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
	out.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	return out, nil
}
