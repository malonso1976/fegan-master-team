package fegan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	pdf "github.com/giraffesyo/pdf"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"golang.org/x/text/unicode/norm"
)

type Catalog struct {
	Clubs       []domain.Club `json:"clubs"`
	SeasonStart int           `json:"season_start"`
	SourceURL   string        `json:"source_url"`
	Warnings    []string      `json:"warnings"`
}

func extractPDF(ctx context.Context, b []byte) (*pdf.Document, error) {
	if len(b) > 20<<20 || !bytes.HasPrefix(b, []byte("%PDF")) {
		return nil, errors.New("PDF no válido o superior a 20 MiB")
	}
	doc, e := pdf.Extract(ctx, bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return nil, fmt.Errorf("no se pudo leer el PDF: %w", e)
	}
	if len(doc.Pages) > 400 {
		return nil, errors.New("PDF de más de 400 páginas")
	}
	return doc, nil
}
func normalize(s string) string {
	var out strings.Builder
	for _, r := range norm.NFD.String(strings.ToUpper(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}
func pdfRows(p pdf.Page) []pdf.Page {
	groups := map[int][]pdf.Glyph{}
	for _, g := range p.Glyphs {
		y := int(math.Round(g.Y))
		groups[y] = append(groups[y], g)
	}
	keys := []int{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	out := []pdf.Page{}
	for _, k := range keys {
		row := p
		row.Glyphs = groups[k]
		out = append(out, row)
	}
	return out
}
func column(p pdf.Page, low, high float64) string {
	row := p
	row.Glyphs = nil
	for _, g := range p.Glyphs {
		if g.X >= low && g.X < high {
			row.Glyphs = append(row.Glyphs, g)
		}
	}
	return strings.TrimSpace(row.Text())
}

var seasonPattern = regexp.MustCompile(`Tempada\s+(20\d{2})/(\d{2})`)
var codePattern = regexp.MustCompile(`^[0-9P][0-9]{4}$`)

func ParseClubsPDF(ctx context.Context, b []byte, source string) (Catalog, error) {
	out := Catalog{Clubs: []domain.Club{}, SourceURL: source, Warnings: []string{}}
	doc, e := extractPDF(ctx, b)
	if e != nil {
		return out, e
	}
	season := seasonPattern.FindStringSubmatch(doc.Text())
	if len(season) != 3 {
		return out, errors.New("catálogo sin temporada reconocible")
	}
	out.SeasonStart, _ = strconv.Atoi(season[1])
	seen := map[string]bool{}
	for _, page := range doc.Pages {
		if !strings.Contains(page.Text(), "Nome medio") {
			continue
		}
		width := page.MediaBox.MaxX - page.MediaBox.MinX
		scale := width / 595.28
		for _, row := range pdfRows(page) {
			code := column(row, 56*scale, 87*scale)
			if !codePattern.MatchString(code) {
				continue
			}
			c := domain.Club{ID: code, Name: column(row, 87*scale, 345*scale), MediumName: column(row, 345*scale, 487*scale), ShortName: column(row, 487*scale, 565*scale), SourceURL: source, SeasonStart: out.SeasonStart}
			if c.Name == "" || c.MediumName == "" || c.ShortName == "" || seen[c.ID] {
				return out, errors.New("columnas del catálogo de clubes no reconocidas")
			}
			if e = c.Validate(); e != nil {
				return out, e
			}
			seen[c.ID] = true
			out.Clubs = append(out.Clubs, c)
		}
	}
	if len(out.Clubs) == 0 {
		return out, errors.New("no se encontraron clubes válidos en el catálogo")
	}
	sort.Slice(out.Clubs, func(i, j int) bool { return out.Clubs[i].ID < out.Clubs[j].ID })
	return out, nil
}

type RelayPDFResult struct {
	Data     domain.Data
	Warnings []string
}

var relayHeading = regexp.MustCompile(`Prueba\s+\d+[, ]+\s*(Masc\.|Fem\.|Mixto)[, ]+\s*4\s*x\s*(50|100|200)m\s+(Libre|Estilos)`)
var relayCategory = regexp.MustCompile(`(?:^|,\s*)\+(80|100|120|160|200|240|280|320)(?:\s|$|\()`)
var resultLine = regexp.MustCompile(`^(\d+)\.\s+(.+?)\s+(\d+:\d{2}[.,]\d{2})\s+\d+(?:[.,]\d+)?(?:\s+.*)?$`)
var meetPool = regexp.MustCompile(`(?:P|\(|\|\s*)(25|50)\s*(?:Elec|Elect|m|\))`)
var resultDate = regexp.MustCompile(`^(\d{2})/(\d{2})/(20\d{2})\s+Resultados`)

func ParseRelayPDF(ctx context.Context, b []byte, source, date, location string, clubs []domain.Club, aliases map[string]string) (RelayPDFResult, error) {
	out := RelayPDFResult{Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}, Warnings: []string{}}
	doc, e := extractPDF(ctx, b)
	if e != nil {
		return out, e
	}
	known := map[string]domain.Club{}
	names := map[string]string{}
	for _, c := range clubs {
		known[c.ID] = c
		for _, name := range []string{c.Name, c.ShortName, c.MediumName} {
			if name != "" {
				names[normalize(name)] = c.ID
			}
		}
	}
	for name, id := range aliases {
		names[normalize(name)] = id
	}
	event, gender, category, pool := "", "", 0, 0
	foundRelay := false
	skipped := map[string]bool{}
	usedClubs := map[string]bool{}
	for _, page := range doc.Pages {
		if e := ctx.Err(); e != nil {
			return out, e
		}
		for _, line := range strings.Split(page.Text(), "\n") {
			line = strings.TrimSpace(line)
			if m := meetPool.FindStringSubmatch(line); len(m) > 0 {
				pool, _ = strconv.Atoi(m[1])
			}
			if strings.Contains(line, "Prueba ") {
				m := relayHeading.FindStringSubmatch(line)
				if m == nil {
					event = ""
					continue
				}
				foundRelay = true
				gender = map[string]string{"Masc.": "M", "Fem.": "F", "Mixto": "X"}[m[1]]
				style := "L"
				if m[3] == "Estilos" {
					style = "S"
				}
				event = "4x" + m[2] + style
				category = 0
			}
			if event == "" {
				continue
			}
			if m := resultDate.FindStringSubmatch(line); len(m) > 0 {
				date = m[3] + "-" + m[2] + "-" + m[1]
			}
			if m := relayCategory.FindStringSubmatch(line); len(m) > 0 {
				category, _ = strconv.Atoi(m[1])
			}
			m := resultLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			if category == 0 || pool == 0 {
				return out, errors.New("resultado de relevo sin categoría o piscina reconocible")
			}
			// Club is the rightmost name before the time; team labels may contain spaces.
			clubID := ""
			bestLength := 0
			name := normalize(m[2])
			for alias, id := range names {
				if strings.HasSuffix(name, alias) && len(alias) > bestLength {
					clubID = id
					bestLength = len(alias)
				}
			}
			if clubID == "" {
				if !skipped[m[2]] {
					out.Warnings = append(out.Warnings, "Club sin código verificable en PDF: "+m[2])
					skipped[m[2]] = true
				}
				continue
			}
			cs, e := domain.ParseTime(m[3])
			if e != nil {
				return out, e
			}
			mark := domain.Mark{ClubID: clubID, Event: event, Pool: pool, Chrono: "E", Date: date, Location: location, Centiseconds: cs, Source: "fegan-pdf", SourceURL: source, RetrievedAt: time.Now().UTC().Format(time.RFC3339), Verified: true, Gender: gender, RelayCategory: category}
			if e = mark.Validate(); e != nil {
				return out, e
			}
			out.Data.Marks = append(out.Data.Marks, mark)
			if !usedClubs[clubID] {
				c, ok := known[clubID]
				if !ok {
					c = domain.Club{ID: clubID, Name: m[2]}
				}
				out.Data.Clubs = append(out.Data.Clubs, c)
				usedClubs[clubID] = true
			}
		}
	}
	if !foundRelay {
		return out, errors.New("PDF sin pruebas de relevos reconocibles")
	}
	return out, nil
}
