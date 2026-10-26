package fegan

import (
	"fmt"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"sort"
	"time"
)

func RivalQueries(d domain.Data, club string, season int, now time.Time) []Query {
	members := map[string]domain.Swimmer{}
	for _, s := range d.Swimmers {
		if s.Member && s.ClubID == club {
			members[s.License] = s
		}
	}
	from, to := now.AddDate(-2, 0, 0).Format("2006-01-02"), now.Format("2006-01-02")
	queries := map[string]Query{}
	for _, m := range d.Marks {
		s, ok := members[m.License]
		if !ok || domain.Relay(m.Event) || m.Date < from || m.Date > to {
			continue
		}
		low, high := domain.IndividualGroup(domain.SeasonAge(season, s.BirthYear))
		if low == 0 {
			continue
		}
		birthFrom := 1900
		if high != 0 {
			birthFrom = season + 1 - high
		}
		q := Query{TopTen: true, Event: m.Event, Gender: s.Gender, Pool: m.Pool, Chrono: m.Chrono, From: from, To: to, BirthFrom: birthFrom, BirthTo: season + 1 - low}
		queries[fmt.Sprintf("%+v", q)] = q
	}
	keys := []string{}
	for key := range queries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := []Query{}
	for _, key := range keys {
		out = append(out, queries[key])
	}
	return out
}

func topTen(d domain.Data) domain.Data {
	best := map[string]domain.Mark{}
	for _, m := range d.Marks {
		old, ok := best[m.License]
		if !ok || m.Centiseconds < old.Centiseconds {
			best[m.License] = m
		}
	}
	marks := []domain.Mark{}
	for _, m := range best {
		marks = append(marks, m)
	}
	sort.Slice(marks, func(i, j int) bool {
		if marks[i].Centiseconds != marks[j].Centiseconds {
			return marks[i].Centiseconds < marks[j].Centiseconds
		}
		return marks[i].License < marks[j].License
	})
	if len(marks) > 10 {
		marks = marks[:10]
	}
	licenses := map[string]bool{}
	clubs := map[string]bool{}
	for _, m := range marks {
		licenses[m.License] = true
		clubs[m.ClubID] = true
	}
	out := domain.Data{Marks: marks, Swimmers: []domain.Swimmer{}, Clubs: []domain.Club{}}
	for _, s := range d.Swimmers {
		if licenses[s.License] {
			out.Swimmers = append(out.Swimmers, s)
		}
	}
	for _, c := range d.Clubs {
		if clubs[c.ID] {
			out.Clubs = append(out.Clubs, c)
		}
	}
	return out
}
