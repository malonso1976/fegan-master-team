package fegan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marianoao/fegan-master-team/internal/domain"
	"golang.org/x/net/html"
)

const personalPath = "/marks/best_results_person"

var personalIdentity = regexp.MustCompile(`^\[?([0-9]+)\]?\s*-\s*(.+)\s+\((\d{4})\)$`)

func personalClubLabels(body []byte) (map[string]string, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	walk(root, func(n *html.Node) {
		if n.Data == "select" && attr(n, "name") == "combo_club" {
			walk(n, func(o *html.Node) {
				if o.Data == "option" && domain.ValidID(attr(o, "value")) {
					labels[text(o)] = attr(o, "value")
				}
			})
		}
	})
	if len(labels) == 0 {
		return nil, errors.New("catálogo de códigos de clubes no reconocido para las marcas personales")
	}
	return labels, nil
}

func parsePersonal(body []byte, q Query, labels map[string]string, source string) (domain.Data, error) {
	out := domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	var identity []string
	var resultTable *html.Node
	formOK := false
	walk(root, func(n *html.Node) {
		if n.Data == "input" && attr(n, "name") == "input_license" {
			formOK = true
		}
		if n.Data == "h3" {
			if m := personalIdentity.FindStringSubmatch(text(n)); m != nil {
				identity = m
			}
		}
		if n.Data == "table" {
			var headers []string
			walk(n, func(c *html.Node) {
				if c.Data == "th" {
					headers = append(headers, text(c))
				}
			})
			if strings.Join(headers, "|") == "Club|Event|Mark|C.P.|Date|Location|Split" {
				resultTable = n
			}
		}
	})
	if !formOK {
		return out, errors.New("formulario personal FEGAN no reconocido")
	}
	if resultTable == nil && strings.Contains(text(root), "Non se atopou ningún resultado.") {
		return out, nil
	}
	if len(identity) != 4 || identity[1] != q.License {
		return out, errors.New("identidad de la consulta personal no verificable")
	}
	year, err := strconv.Atoi(identity[3])
	if err != nil {
		return out, errors.New("nacimiento personal no reconocible")
	}
	if q.BirthFrom != 0 && (year < q.BirthFrom || year > q.BirthTo) {
		return out, errOutsideFilters
	}
	if resultTable == nil {
		return out, errors.New("tabla de marcas personales FEGAN no reconocida")
	}
	rows := [][]string{}
	walk(resultTable, func(n *html.Node) {
		if n.Data == "tr" {
			var cells []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Data == "td" {
					cells = append(cells, text(c))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
		}
	})
	if len(rows) > 128 {
		return out, errors.New("demasiadas mejores marcas personales; revisar el formato")
	}
	clubs := map[string]bool{}
	for _, c := range rows {
		if len(c) != 7 {
			return out, errors.New("columnas personales FEGAN distintas del contrato")
		}
		if domain.Relay(c[1]) || (q.Event != "all" && c[1] != q.Event) {
			continue
		}
		if !domain.ValidEvent(c[1]) {
			return out, errors.New("prueba personal no reconocida")
		}
		if c[3] != fmt.Sprintf("%s%d", q.Chrono, q.Pool) || c[6] != "N" {
			return out, errOutsideFilters
		}
		club, ok := labels[c[0]]
		if !ok {
			return out, errors.New("club personal sin código verificable")
		}
		if q.ClubID != "" && club != q.ClubID {
			continue
		}
		date, err := time.Parse("20060102", c[4])
		if err != nil {
			return out, errors.New("fecha de marca personal no válida")
		}
		day := date.Format("2006-01-02")
		if day < q.From || day > q.To {
			return out, errOutsideFilters
		}
		cs, err := domain.ParseTime(c[2])
		if err != nil {
			return out, errors.New("tiempo de marca personal no válido")
		}
		swimmer := domain.Swimmer{License: q.License, Name: strings.TrimSpace(identity[2]), BirthYear: year, Gender: q.Gender, ClubID: club, Member: q.Member, Available: q.Member}
		if err = swimmer.Validate(); err != nil {
			return out, errors.New("identidad personal FEGAN no válida")
		}
		if len(out.Swimmers) == 0 {
			out.Swimmers = append(out.Swimmers, swimmer)
		}
		mark := domain.Mark{License: q.License, ClubID: club, Event: c[1], Pool: q.Pool, Chrono: q.Chrono, Date: day, Centiseconds: cs, Location: c[5], Source: "fegan", SourceURL: source, Verified: true, Gender: q.Gender, RetrievedAt: time.Now().UTC().Format(time.RFC3339)}
		if err = mark.Validate(); err != nil {
			return out, err
		}
		out.Marks = append(out.Marks, mark)
		if !clubs[club] {
			out.Clubs = append(out.Clubs, domain.Club{ID: club, Name: c[0]})
			clubs[club] = true
		}
	}
	return out, nil
}

// PersonalPreview queries each timing/pool independently so a faster timing never
// hides another usable reference. This endpoint supplies bests, not complete history.
func (c *Client) PersonalPreview(ctx context.Context, q Query) (Preview, error) {
	if q.License == "" || (q.Gender != "M" && q.Gender != "F") {
		return Preview{}, errors.New("la consulta personal requiere licencia y categoría M/F conocida")
	}
	queries, err := ExpandQuery(q)
	if err != nil {
		return Preview{}, err
	}
	catalog, err := c.download(ctx, c.DocumentsBase+"/marks/")
	if err != nil {
		return Preview{}, err
	}
	labels, err := personalClubLabels(catalog)
	if err != nil {
		return Preview{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := "personal:" + fmt.Sprintf("%+v", q)
	if entry, ok := c.cache[key]; ok && time.Since(entry.at) < 15*time.Minute {
		p := entry.result
		p.Cached = true
		return p, nil
	}
	out := Preview{Query: q, Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}, Warnings: []string{"Complemento: mejores marcas personales publicadas por FEGAN, sin sustituir el historial ni convertir tiempos."}}
	clubs := map[string]domain.Club{}
	people := map[string]domain.Swimmer{}
	for _, v := range queries {
		if delay := time.Until(c.last.Add(time.Second)); delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return Preview{}, ctx.Err()
			case <-timer.C:
			}
		}
		values := url.Values{"combo_estament": {"MASTE"}, "input_license": {v.License}, "input_from_date": {v.From}, "input_to_date": {v.To}, "combo_pool": {strconv.Itoa(v.Pool)}, "combo_chrono": {v.Chrono}, "combo_split": {"N"}}
		source := c.DocumentsBase + personalPath
		req, err := http.NewRequestWithContext(ctx, "POST", source, strings.NewReader(values.Encode()))
		if err != nil {
			return Preview{}, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "fegan-master-team/0.1 (local personal results manager)")
		c.last = time.Now()
		out.Requests++
		response, err := c.HTTP.Do(req)
		if err != nil {
			return Preview{}, errors.New("no se pudo consultar el formulario personal FEGAN")
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20+1))
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || len(body) > 4<<20 {
			return Preview{}, errors.New("respuesta personal FEGAN no válida o demasiado grande")
		}
		data, err := parsePersonal(body, v, labels, source)
		if err != nil {
			return Preview{}, err
		}
		out.Data.Marks = append(out.Data.Marks, data.Marks...)
		for _, club := range data.Clubs {
			clubs[club.ID] = club
		}
		for _, person := range data.Swimmers {
			if old, ok := people[person.License]; ok && (old.BirthYear != person.BirthYear || old.Gender != person.Gender) {
				return Preview{}, errors.New("identidad personal incompatible entre consultas")
			}
			people[person.License] = person
		}
	}
	for _, club := range clubs {
		out.Data.Clubs = append(out.Data.Clubs, club)
	}
	for _, person := range people {
		out.Data.Swimmers = append(out.Data.Swimmers, person)
	}
	out.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	if len(c.cache) >= 8 {
		c.cache = map[string]cacheEntry{}
	}
	c.cache[key] = cacheEntry{out, time.Now()}
	return out, nil
}
