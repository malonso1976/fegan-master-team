// Package fegan adapts the public marks search. It never enables weighted conversions.
package fegan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/marianoao/fegan-master-team/internal/domain"
	"golang.org/x/net/html"
)

var errOutsideFilters = errors.New("resultado FEGAN fuera de los filtros solicitados")

const endpoint = "https://www.fegan.org/marks/"

type Query struct {
	TopTen    bool   `json:"-"`
	ClubID    string `json:"club_id"`
	License   string `json:"license"`
	Event     string `json:"event"`
	Gender    string `json:"gender"`
	Pool      int    `json:"pool"`
	Chrono    string `json:"chrono"`
	From      string `json:"from"`
	To        string `json:"to"`
	BirthFrom int    `json:"birth_from"`
	BirthTo   int    `json:"birth_to"`
	Member    bool   `json:"member"`
}

func (q Query) Validate() error {
	f := domain.Filter{From: q.From, To: q.To, SeasonStart: domain.SeasonStartOn(time.Now()), Pool: q.Pool, Chrono: q.Chrono}
	if e := f.Validate(); e != nil {
		return e
	}
	if !domain.ValidEvent(q.Event) || domain.Relay(q.Event) || (q.Gender != "M" && q.Gender != "F") || (q.ClubID != "" && !domain.ValidID(q.ClubID)) || (q.License != "" && !domain.ValidID(q.License)) {
		return errors.New("consulta FEGAN no válida; importar relevos manualmente")
	}
	if q.BirthFrom != 0 || q.BirthTo != 0 {
		if q.BirthFrom < 1900 || q.BirthTo < q.BirthFrom || q.BirthTo > time.Now().Year() {
			return errors.New("intervalo de nacimiento no válido")
		}
	}
	if q.ClubID == "" && q.License == "" && q.BirthFrom == 0 {
		return errors.New("acota por club, licencia o años de nacimiento")
	}
	if q.Member && q.ClubID == "" {
		return errors.New("para incorporar miembros, especifica el club")
	}
	return nil
}

type Preview struct {
	Query       Query       `json:"query"`
	Data        domain.Data `json:"data"`
	Requests    int         `json:"requests"`
	Cached      bool        `json:"cached"`
	RetrievedAt string      `json:"retrieved_at"`
	Warnings    []string    `json:"warnings"`
}
type cacheEntry struct {
	result Preview
	at     time.Time
}
type Client struct {
	HTTP          *http.Client
	URL           string
	mu            sync.Mutex
	last          time.Time
	cache         map[string]cacheEntry
	documentDates map[string]time.Time
	DocumentsBase string
	documents     map[string][]byte
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || r.URL.Host != "www.fegan.org" || len(via) > 3 {
			return errors.New("redirección FEGAN no permitida")
		}
		return nil
	}}, URL: endpoint, cache: map[string]cacheEntry{}, DocumentsBase: "https://www.fegan.org", documents: map[string][]byte{}, documentDates: map[string]time.Time{}}
}
func (c *Client) Preview(ctx context.Context, q Query) (Preview, error) {
	if q.Event != "all" {
		p, e := c.preview(ctx, q)
		if q.TopTen && errors.Is(e, errOutsideFilters) {
			original := q
			q.TopTen = false
			p, e = c.preview(ctx, q)
			if e == nil {
				p.Data = topTen(p.Data)
				p.Query = original
				p.Warnings = append(p.Warnings, "Se consultó el historial completo para obtener diez referencias compatibles: la consulta top diez devolvió filas fuera de filtro.")
			}
		}
		return p, e
	}
	result := Preview{Query: q, Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}, Warnings: []string{"Cobertura: todas las pruebas individuales del catálogo FEGAN para los filtros indicados; no es una clasificación oficial."}, Cached: true}
	clubs := map[string]domain.Club{}
	swimmers := map[string]domain.Swimmer{}
	for _, event := range domain.Events {
		if domain.Relay(event) {
			continue
		}
		q.Event = event
		p, e := c.preview(ctx, q)
		if e != nil {
			return Preview{}, fmt.Errorf("%s: %w", event, e)
		}
		result.Requests += p.Requests
		result.Warnings = append(result.Warnings, p.Warnings...)
		result.Cached = result.Cached && p.Cached
		result.Data.Marks = append(result.Data.Marks, p.Data.Marks...)
		for _, v := range p.Data.Clubs {
			clubs[v.ID] = v
		}
		for _, v := range p.Data.Swimmers {
			if old, ok := swimmers[v.License]; ok && (old.BirthYear != v.BirthYear || old.Gender != v.Gender) {
				return Preview{}, errors.New("conflicto de identidad entre pruebas")
			}
			swimmers[v.License] = v
		}
	}
	for _, v := range clubs {
		result.Data.Clubs = append(result.Data.Clubs, v)
	}
	for _, v := range swimmers {
		result.Data.Swimmers = append(result.Data.Swimmers, v)
	}
	result.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	return result, nil
}
func (c *Client) preview(ctx context.Context, q Query) (Preview, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := Preview{Query: q, Warnings: []string{"Comparativa limitada a los datos importados; no es una clasificación oficial."}}
	if e := q.Validate(); e != nil {
		return out, e
	}
	key := fmt.Sprintf("%+v", q)
	if entry, ok := c.cache[key]; ok && time.Since(entry.at) < 15*time.Minute {
		p := entry.result
		p.Cached = true
		return p, nil
	}
	out.Data = domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}
	clubs := map[string]domain.Club{}
	people := map[string]domain.Swimmer{}
	var fetch func(string, string) error
	fetch = func(from, to string) error {
		if out.Requests >= 64 {
			return errors.New("consulta demasiado amplia; divide el intervalo de fechas (máximo 64 peticiones)")
		}
		wait := time.Until(c.last.Add(time.Second))
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
		values := url.Values{"combo_estament": {"MASTE"}, "combo_event": {q.Event}, "combo_gender": {q.Gender}, "combo_pool": {strconv.Itoa(q.Pool)}, "combo_chrono": {q.Chrono}, "combo_split": {"N"}, "combo_club": {q.ClubID}, "input_license": {q.License}, "input_from_date": {from}, "input_to_date": {to}, "combo_show_first": {"100"}}
		if q.TopTen {
			values.Set("combo_show_first", "10")
			values.Set("input_best_mark", "on")
		}
		if q.BirthFrom != 0 {
			values.Set("input_from_year", strconv.Itoa(q.BirthFrom))
			values.Set("input_to_year", strconv.Itoa(q.BirthTo))
		}
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, strings.NewReader(values.Encode()))
		if e != nil {
			return e
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "fegan-master-team/0.1 (local personal results manager)")
		c.last = time.Now()
		out.Requests++
		resp, e := c.HTTP.Do(req)
		if e != nil {
			return fmt.Errorf("no se pudo consultar FEGAN: %w", e)
		}
		body, e := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024+1))
		resp.Body.Close()
		if e != nil {
			return e
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("FEGAN respondió HTTP %d", resp.StatusCode)
		}
		if len(body) > 4*1024*1024 {
			return errors.New("respuesta FEGAN demasiado grande")
		}
		parsed, e := parseResults(strings.NewReader(string(body)), q)
		if e != nil {
			return e
		}
		data := parsed.Data
		if !q.TopTen && parsed.Rows >= 100 {
			if from == to {
				return errors.New("FEGAN devuelve 100 resultados en un solo día; no se puede garantizar la importación completa")
			}
			a, _ := time.Parse("2006-01-02", from)
			b, _ := time.Parse("2006-01-02", to)
			mid := a.AddDate(0, 0, int(b.Sub(a).Hours()/24)/2)
			if e = fetch(from, mid.Format("2006-01-02")); e != nil {
				return e
			}
			return fetch(mid.AddDate(0, 0, 1).Format("2006-01-02"), to)
		}
		if parsed.SkippedBirthYears > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("Se omitieron filas sin año de nacimiento: %d (%s, %d m, %s). Las demás marcas válidas se conservan.", parsed.SkippedBirthYears, q.Event, q.Pool, q.Chrono))
		}
		if q.TopTen {
			data = topTen(data)
		}
		for _, club := range data.Clubs {
			clubs[club.ID] = club
		}
		for _, s := range data.Swimmers {
			if prior, ok := people[s.License]; ok && (prior.BirthYear != s.BirthYear || prior.Gender != s.Gender) {
				return errors.New("identidades incompatibles en FEGAN")
			}
			people[s.License] = s
		}
		out.Data.Marks = append(out.Data.Marks, data.Marks...)
		return nil
	}
	if e := fetch(q.From, q.To); e != nil {
		return Preview{}, e
	}
	for _, v := range clubs {
		out.Data.Clubs = append(out.Data.Clubs, v)
	}
	for _, v := range people {
		out.Data.Swimmers = append(out.Data.Swimmers, v)
	}
	sort.Slice(out.Data.Swimmers, func(i, j int) bool { return out.Data.Swimmers[i].License < out.Data.Swimmers[j].License })
	out.RetrievedAt = time.Now().UTC().Format(time.RFC3339)
	if len(c.cache) >= 8 {
		c.cache = map[string]cacheEntry{}
	}
	c.cache[key] = cacheEntry{out, time.Now()}
	return out, nil
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	s := ""
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		s += text(ch)
	}
	return strings.TrimSpace(s)
}
func walk(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		walk(ch, fn)
	}
}

type parsedResults struct {
	Data              domain.Data
	Rows              int
	SkippedBirthYears int
}

func Parse(r io.Reader, q Query) (domain.Data, error) {
	result, err := parseResults(r, q)
	return result.Data, err
}
func parseResults(r io.Reader, q Query) (parsedResults, error) {
	result := parsedResults{Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}}
	d := &result.Data
	root, e := html.Parse(r)
	if e != nil {
		return result, e
	}
	clubs := map[string]domain.Club{}
	labels := map[string]string{}
	var table *html.Node
	formOK := false
	walk(root, func(n *html.Node) {
		if n.Data == "select" && attr(n, "name") == "combo_club" {
			formOK = true
			walk(n, func(o *html.Node) {
				if o.Data == "option" && attr(o, "value") != "" {
					labels[text(o)] = attr(o, "value")
				}
			})
		}
		if attr(n, "id") == "dvData" {
			table = n
		}
	})
	if !formOK {
		return result, errors.New("formato FEGAN no reconocido: falta el catálogo de clubes")
	}
	if table == nil && strings.Contains(text(root), "Non se atopou ningún resultado.") {
		return result, nil
	}
	if table == nil {
		return result, errors.New("FEGAN no devolvió una tabla de resultados; consulta vacía o cambio de formato")
	}
	var headers []string
	var rows [][]string
	walk(table, func(n *html.Node) {
		if n.Data != "tr" {
			return
		}
		var cells []string
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Data == "th" {
				headers = append(headers, text(ch))
			}
			if ch.Data == "td" {
				cells = append(cells, text(ch))
			}
		}
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	})
	expected := []string{"License", "Name", "Year", "Club", "Event", "Mark", "C.P.", "Date", "Location", "Split"}
	if strings.Join(headers, "|") != strings.Join(expected, "|") {
		return result, errors.New("columnas FEGAN diferentes al contrato conocido")
	}
	result.Rows = len(rows)
	seen := map[string]bool{}
	for i, cells := range rows {
		if len(cells) != 10 {
			return result, fmt.Errorf("fila FEGAN %d con columnas inesperadas", i+1)
		}
		c := cells
		if strings.TrimSpace(c[2]) == "" {
			result.SkippedBirthYears++
			continue
		}
		club, ok := labels[c[3]]
		if !ok {
			return result, errors.New("club de un resultado no encontrado en el catálogo FEGAN")
		}
		year, e := strconv.Atoi(c[2])
		if e != nil {
			return result, e
		}
		cs, e := domain.ParseTime(c[5])
		if e != nil {
			return result, fmt.Errorf("marca FEGAN no válida: fila %d", i+1)
		}
		date, e := time.Parse("20060102", c[7])
		if e != nil {
			return result, e
		}
		if c[6] != fmt.Sprintf("%s%d", q.Chrono, q.Pool) || c[4] != q.Event || c[9] != "N" || (q.License != "" && c[0] != q.License) || (q.ClubID != "" && club != q.ClubID) || (q.BirthFrom > 0 && (year < q.BirthFrom || year > q.BirthTo)) || date.Format("2006-01-02") < q.From || date.Format("2006-01-02") > q.To {
			return result, errOutsideFilters
		}
		s := domain.Swimmer{License: c[0], Name: c[1], BirthYear: year, Gender: q.Gender, ClubID: club, Member: q.Member, Available: q.Member}
		if e = s.Validate(); e != nil {
			return result, e
		}
		if !seen[s.License] {
			d.Swimmers = append(d.Swimmers, s)
			seen[s.License] = true
		}
		clubs[club] = domain.Club{ID: club, Name: c[3]}
		m := domain.Mark{License: c[0], ClubID: club, Event: c[4], Pool: q.Pool, Chrono: q.Chrono, Date: date.Format("2006-01-02"), Location: c[8], Centiseconds: cs, Source: "fegan", SourceURL: endpoint, RetrievedAt: time.Now().UTC().Format(time.RFC3339), Verified: true, Gender: q.Gender}
		if e = m.Validate(); e != nil {
			return result, e
		}
		d.Marks = append(d.Marks, m)
	}
	for _, v := range clubs {
		d.Clubs = append(d.Clubs, v)
	}
	return result, nil
}

func (c *Client) RefreshCaches(marks, documents bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if marks {
		c.cache = map[string]cacheEntry{}
	}
	if documents {
		c.documents = map[string][]byte{}
		c.documentDates = map[string]time.Time{}
	}
}
