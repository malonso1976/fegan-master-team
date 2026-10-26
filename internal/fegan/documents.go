package fegan

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/marianoao/fegan-master-team/internal/domain"
	"golang.org/x/net/html"
)

func (c *Client) download(ctx context.Context, address string) ([]byte, error) {
	base, _ := url.Parse(c.DocumentsBase)
	u, e := url.Parse(address)
	if e != nil || u.Host != base.Host || u.Scheme != base.Scheme {
		return nil, errors.New("fuente documental fuera de FEGAN")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.documents[address]; ok && time.Since(c.documentDates[address]) < time.Hour {
		return b, nil
	}
	if delay := time.Until(c.last.Add(time.Second)); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	req, e := http.NewRequestWithContext(ctx, "GET", address, nil)
	if e != nil {
		return nil, e
	}
	c.last = time.Now()
	resp, e := c.HTTP.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, address)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 32<<20 {
		return nil, errors.New("documento superior a 32 MiB")
	}
	if c.documents == nil {
		c.documents = map[string][]byte{}
	}
	if len(c.documents) >= 16 {
		c.documents = map[string][]byte{}
	}
	if c.documentDates == nil {
		c.documentDates = map[string]time.Time{}
	}
	c.documentDates[address] = time.Now()
	c.documents[address] = b
	return b, nil
}
func absolute(base, ref string) string {
	u, e := url.Parse(base)
	if e != nil {
		return ""
	}
	r, e := url.Parse(ref)
	if e != nil {
		return ""
	}
	resolved := u.ResolveReference(r)
	if (resolved.Host == "www.fegan.org" || resolved.Host == "fegan.org") && (resolved.Scheme == "http" || resolved.Scheme == "https") {
		resolved.Host = "www.fegan.org"
		resolved.Scheme = "https"
	}
	return resolved.String()
}
func (c *Client) ClubCatalog(ctx context.Context, start int) (Catalog, error) {
	address := fmt.Sprintf("%s/fegan/tprd/%02d%02d/xeral/clubs_activos.pdf", c.DocumentsBase, start%100, (start+1)%100)
	b, e := c.download(ctx, address)
	if e != nil {
		return Catalog{}, e
	}
	out, e := ParseClubsPDF(ctx, b, address)
	if e == nil && out.SeasonStart != start {
		return Catalog{}, errors.New("la temporada del catálogo no coincide con la solicitada")
	}
	return out, e
}

type Championship struct {
	Date        string
	Name        string
	PDF         string
	ZIP         string
	Web         string
	SeasonStart int
}

var archivePath = regexp.MustCompile(`/fegan/tprd/(\d{4})/index\.htm`)
var championshipDate = regexp.MustCompile(`20\d\d-\d\d-\d\d`)

func championships(b []byte, base string) []Championship {
	root, e := html.Parse(bytes.NewReader(b))
	if e != nil {
		return nil
	}
	out := []Championship{}
	walk(root, func(n *html.Node) {
		if n.Data != "tr" {
			return
		}
		label := text(n)
		normalized := normalize(label)
		if !strings.Contains(normalized, "MAST") || (!strings.Contains(normalized, "INVERNO") && !strings.Contains(normalized, "VERAN")) {
			return
		}
		date := championshipDate.FindString(label)
		if date == "" {
			return
		}
		parsed, _ := time.Parse("2006-01-02", date)
		name := "Campeonato Gallego Máster de Verano"
		if strings.Contains(normalized, "INVERNO") {
			name = "Campeonato Gallego Máster de Invierno"
		}
		entry := Championship{Date: date, Name: name, SeasonStart: domain.SeasonStartOn(parsed)}
		walk(n, func(a *html.Node) {
			if a.Data != "a" {
				return
			}
			ref := attr(a, "href")
			u := absolute(base, ref)
			switch {
			case strings.HasSuffix(strings.ToLower(ref), ".pdf"):
				entry.PDF = u
			case strings.HasSuffix(strings.ToLower(ref), ".zip"):
				entry.ZIP = u
			case strings.Contains(ref, "index.html"):
				entry.Web = u
			}
		})
		if entry.PDF != "" || entry.ZIP != "" || entry.Web != "" {
			out = append(out, entry)
		}
	})
	return out
}
func (c *Client) Championships(ctx context.Context, now time.Time) ([]Championship, error) {
	entryURL := c.DocumentsBase + "/post/57"
	b, e := c.download(ctx, entryURL)
	if e != nil {
		return nil, e
	}
	all := championships(b, entryURL)
	archiveURL := c.DocumentsBase + "/post/20"
	archive, e := c.download(ctx, archiveURL)
	if e != nil {
		return nil, e
	}
	root, e := html.Parse(bytes.NewReader(archive))
	if e != nil {
		return nil, e
	}
	archives := []string{}
	walk(root, func(n *html.Node) {
		if n.Data == "a" {
			ref := absolute(archiveURL, attr(n, "href"))
			if archivePath.MatchString(ref) {
				archives = append(archives, ref)
			}
		}
	})
	sort.Sort(sort.Reverse(sort.StringSlice(archives)))
	seasons := map[int]bool{}
	for _, ch := range all {
		seasons[ch.SeasonStart] = true
	}
	for _, ref := range archives {
		if len(seasons) >= 2 {
			break
		}
		b, e = c.download(ctx, ref)
		if e != nil {
			return nil, e
		}
		for _, ch := range championships(b, ref) {
			if ch.Date <= now.Format("2006-01-02") {
				all = append(all, ch)
				seasons[ch.SeasonStart] = true
			}
		}
	}
	years := []int{}
	for year := range seasons {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	if len(years) < 2 {
		return nil, errors.New("no se localizaron campeonatos de dos temporadas en las fuentes oficiales")
	}
	valid := map[int]bool{years[0]: true, years[1]: true}
	seen := map[string]bool{}
	out := []Championship{}
	for _, ch := range all {
		key := ch.Date
		if valid[ch.SeasonStart] && !seen[key] && ch.Date <= now.Format("2006-01-02") {
			seen[key] = true
			out = append(out, ch)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out, nil
}
func unzipFiles(b []byte) (map[string][]byte, error) {
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return nil, e
	}
	if len(z.File) > 600 {
		return nil, errors.New("ZIP con demasiados archivos")
	}
	out := map[string][]byte{}
	var total uint64
	for _, f := range z.File {
		lower := strings.ToLower(f.Name)
		if !strings.HasSuffix(lower, ".pdf") && !strings.HasSuffix(lower, ".lxf") && !strings.HasSuffix(lower, ".lef") {
			continue
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 20<<20 || total > 80<<20 {
			return nil, errors.New("ZIP excede límites de extracción")
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, 20<<20+1))
		r.Close()
		if e != nil {
			return nil, e
		}
		out[f.Name] = b
	}
	return out, nil
}
func lenexAliases(b []byte) (map[string]string, error) {
	if bytes.HasPrefix(b, []byte("PK")) {
		files, e := unzipFiles(b)
		if e != nil {
			return nil, e
		}
		for name, value := range files {
			if strings.HasSuffix(strings.ToLower(name), ".lef") {
				b = value
				break
			}
		}
	}
	decoder := xml.NewDecoder(bytes.NewReader(b))
	out := map[string]string{}
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "CLUB" {
			continue
		}
		attrs := map[string]string{}
		for _, a := range start.Attr {
			attrs[a.Name.Local] = a.Value
		}
		id := attrs["code"]
		if !domain.ValidID(id) {
			continue
		}
		for _, key := range []string{"name", "shortname"} {
			if name := attrs[key]; name != "" {
				if old, ok := out[name]; ok && old != id {
					return nil, errors.New("alias de club ambiguo en Lenex")
				}
				out[name] = id
			}
		}
	}
	return out, nil
}
func (c *Client) RelayChampionship(ctx context.Context, ch Championship, clubs []domain.Club) (RelayPDFResult, error) {
	out := RelayPDFResult{Data: domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}, Warnings: []string{}}
	aliases := map[string]string{}
	pdfs := map[string][]byte{}
	var problems []string
	// Archives supply official club aliases and complete result PDFs, never extracted to disk.
	if ch.ZIP != "" {
		b, e := c.download(ctx, ch.ZIP)
		if e == nil {
			files, e := unzipFiles(b)
			if e == nil {
				for name, value := range files {
					if strings.HasSuffix(strings.ToLower(name), ".lxf") {
						a, e := lenexAliases(value)
						if e == nil {
							for key, id := range a {
								aliases[key] = id
							}
						}
					}
					if strings.Contains(strings.ToLower(name), "resultados_completos") && strings.HasSuffix(strings.ToLower(name), ".pdf") {
						pdfs[ch.ZIP+"#"+name] = value
					}
				}
			} else {
				problems = append(problems, e.Error())
			}
		} else {
			problems = append(problems, e.Error())
		}
	}
	if len(pdfs) == 0 && ch.PDF != "" {
		if b, e := c.download(ctx, ch.PDF); e == nil {
			pdfs[ch.PDF] = b
		} else {
			problems = append(problems, e.Error())
		}
	}
	if ch.Web != "" {
		b, e := c.download(ctx, ch.Web)
		if e == nil {
			root, _ := html.Parse(bytes.NewReader(b))
			var lxf string
			refs := map[string]bool{}
			walk(root, func(n *html.Node) {
				if n.Data == "a" && strings.HasSuffix(attr(n, "href"), ".lxf") {
					lxf = absolute(ch.Web, attr(n, "href"))
				}
				if n.Data == "tr" && strings.Contains(text(n), "4 x 50m") {
					walk(n, func(a *html.Node) {
						if a.Data == "a" && strings.HasPrefix(path.Base(attr(a, "href")), "ResultList_") && strings.HasSuffix(attr(a, "href"), ".pdf") {
							refs[absolute(ch.Web, attr(a, "href"))] = true
						}
					})
				}
			})
			if lxf != "" {
				if b, e := c.download(ctx, lxf); e == nil {
					if a, e := lenexAliases(b); e == nil {
						for key, id := range a {
							aliases[key] = id
						}
					}
				}
			}
			if len(pdfs) == 0 {
				for ref := range refs {
					if b, e := c.download(ctx, ref); e == nil {
						pdfs[ref] = b
					} else {
						problems = append(problems, e.Error())
					}
				}
			}
		} else {
			problems = append(problems, e.Error())
		}
	}
	if len(pdfs) == 0 {
		return out, fmt.Errorf("sin PDF accesible para %s: %s", ch.Date, strings.Join(problems, "; "))
	}
	refs := []string{}
	for ref := range pdfs {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		p, e := ParseRelayPDF(ctx, pdfs[ref], ref, ch.Date, ch.Name, clubs, aliases)
		if e != nil {
			return out, fmt.Errorf("%s: %w", ref, e)
		}
		out.Data.Clubs = append(out.Data.Clubs, p.Data.Clubs...)
		out.Data.Marks = append(out.Data.Marks, p.Data.Marks...)
		out.Warnings = append(out.Warnings, p.Warnings...)
	}
	if len(out.Data.Marks) == 0 {
		return out, errors.New("no se pudieron vincular marcas de relevos con códigos de club")
	}
	return out, nil
}
