package fegan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func personalQuery() Query {
	return Query{License: "00001", Gender: "M", Event: "all", From: "2021-10-28", To: "2026-10-28", Pool: 50, Chrono: "E"}
}
func TestPersonalContractAndHistoricalClub(t *testing.T) {
	b, err := os.ReadFile("testdata/personal.html")
	if err != nil {
		t.Fatal(err)
	}
	q := personalQuery()
	labels := map[string]string{"CLUBTEST": "00123"}
	d, err := parsePersonal(b, q, labels, "https://www.fegan.org"+personalPath)
	if err != nil || len(d.Marks) != 1 || d.Marks[0].License != "00001" || d.Marks[0].Event != "400L" || d.Marks[0].Centiseconds != 40000 || d.Marks[0].ClubID != "00123" || d.Swimmers[0].BirthYear != 1986 || d.Marks[0].SourceURL != "https://www.fegan.org"+personalPath {
		t.Fatal("personal contract mismatch", err)
	}
	for _, bad := range []string{strings.ReplaceAll(string(b), "00001 -", "99999 -"), strings.ReplaceAll(string(b), "<th>Mark</th>", "<th>Time</th>"), strings.ReplaceAll(string(b), "E50", "M50"), strings.ReplaceAll(string(b), "20230603", "20100603"), strings.ReplaceAll(string(b), "<td>N</td>", "<td>S</td>"), strings.ReplaceAll(string(b), "CLUBTEST", "UNKNOWN")} {
		if _, err = parsePersonal([]byte(bad), q, labels, "source"); err == nil {
			t.Fatal("invalid personal response accepted")
		}
	}
	q.Event = "50L"
	d, err = parsePersonal(b, q, labels, "source")
	if err != nil || len(d.Marks) != 0 {
		t.Fatal("event filter lost", err)
	}
}
func TestPersonalPreviewTimingPartitionsAndCache(t *testing.T) {
	b, _ := os.ReadFile("testdata/personal.html")
	var posts atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, `<select name="combo_club"><option value="00123">CLUBTEST</option></select>`)
			return
		}
		posts.Add(1)
		r.ParseForm()
		for k, v := range map[string]string{"combo_estament": "MASTE", "input_license": "00001", "input_from_date": "2021-10-28", "input_to_date": "2026-10-28", "combo_split": "N"} {
			if r.Form.Get(k) != v {
				t.Errorf("incorrect personal filter %s", k)
			}
		}
		if r.URL.Path != personalPath || r.Form.Get("input_equate") != "" {
			t.Error("wrong endpoint or conversion enabled")
		}
		cp := r.Form.Get("combo_chrono") + r.Form.Get("combo_pool")
		fmt.Fprint(w, strings.ReplaceAll(string(b), "E50", cp))
	}))
	defer remote.Close()
	c := New()
	c.DocumentsBase = remote.URL
	c.HTTP = remote.Client()
	q := personalQuery()
	q.Pool = 0
	q.Chrono = ""
	p, err := c.PersonalPreview(context.Background(), q)
	if err != nil || len(p.Data.Marks) != 4 || p.Requests != 4 {
		t.Fatal("personal partitions", err)
	}
	seen := map[string]bool{}
	for _, m := range p.Data.Marks {
		seen[fmt.Sprint(m.Pool)+m.Chrono] = true
	}
	if len(seen) != 4 {
		t.Fatal("timings or pools lost")
	}
	cached, err := c.PersonalPreview(context.Background(), q)
	if err != nil || !cached.Cached || posts.Load() != 4 {
		t.Fatal("personal cache miss", err)
	}
	merged := p
	if err = mergePersonal(&merged, p); err != nil || len(merged.Data.Marks) != 4 {
		t.Fatal("supplement duplicated results", err)
	}
}
func TestMissingBirthYearKeepsOtherResultsAndRawCount(t *testing.T) {
	b, _ := os.ReadFile("testdata/results.html")
	text := string(b)
	start := strings.Index(text, "<tr><td>")
	end := start + strings.Index(text[start:], "</tr>") + len("</tr>")
	row := text[start:end]
	rows := strings.Replace(row, "<td>1986</td>", "<td></td>", 1) + strings.Repeat(row, 99)
	parsed, err := parseResults(strings.NewReader(strings.Replace(text, row, rows, 1)), query())
	if err != nil || parsed.Rows != 100 || parsed.SkippedBirthYears != 1 || len(parsed.Data.Marks) != 99 {
		t.Fatal("blank identity aborted query or hid truncation", err)
	}
}

func TestBlankBirthYearWarningAndPersonalSupplementOfGeneralHistory(t *testing.T) {
	general, _ := os.ReadFile("testdata/results.html")
	personal, _ := os.ReadFile("testdata/personal.html")
	text := strings.ReplaceAll(string(general), "50L", "400L")
	text = strings.ReplaceAll(text, "26.02", "6:45.00")
	start := strings.Index(text, "<tr><td>")
	end := start + strings.Index(text[start:], "</tr>") + len("</tr>")
	invalid := strings.ReplaceAll(strings.ReplaceAll(text[start:end], "00001", "00002"), "<td>1986</td>", "<td></td>")
	text = strings.Replace(text, "</tbody>", invalid+"</tbody>", 1)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, `<select name="combo_club"><option value="00123">CLUBTEST</option></select>`)
			return
		}
		if r.URL.Path == personalPath {
			fmt.Fprint(w, strings.ReplaceAll(string(personal), "E50", "E25"))
			return
		}
		fmt.Fprint(w, text)
	}))
	defer remote.Close()
	c := New()
	c.URL = remote.URL + "/marks/"
	c.DocumentsBase = remote.URL
	c.HTTP = remote.Client()
	q := personalQuery()
	q.Event = "400L"
	q.Pool = 25
	p, err := c.WidePreview(context.Background(), q)
	if err != nil || len(p.Data.Marks) != 2 {
		t.Fatal("valid historical or personal mark lost", err)
	}
	warning := false
	for _, w := range p.Warnings {
		warning = warning || strings.HasPrefix(w, "Se omitieron filas sin año de nacimiento: 1")
	}
	if !warning {
		t.Fatal("missing birth year omission was hidden")
	}
	dates := map[string]bool{}
	for _, m := range p.Data.Marks {
		dates[m.Date] = true
	}
	if !dates["2026-06-13"] || !dates["2023-06-03"] {
		t.Fatal("personal best replaced history")
	}
}
