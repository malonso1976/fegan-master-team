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
	"time"
)

func query() Query {
	return Query{ClubID: "00123", Event: "50L", Gender: "M", Pool: 25, Chrono: "E", From: "2025-01-01", To: "2026-12-31"}
}
func TestParserContract(t *testing.T) {
	b, e := os.ReadFile("testdata/results.html")
	if e != nil {
		t.Fatal(e)
	}
	d, e := Parse(strings.NewReader(string(b)), query())
	if e != nil || len(d.Marks) != 1 || d.Swimmers[0].License != "00001" || d.Clubs[0].ID != "00123" || d.Marks[0].Centiseconds != 2602 {
		t.Fatal(d, e)
	}
	for _, bad := range []string{strings.ReplaceAll(string(b), "<th>Mark</th>", "<th>Time</th>"), strings.ReplaceAll(string(b), "E25", "E50"), strings.ReplaceAll(string(b), "26.02", "DQ"), "<html>Maintenance</html>"} {
		if _, e = Parse(strings.NewReader(bad), query()); e == nil {
			t.Fatal("invalid contract accepted")
		}
	}
	empty := `<select name="combo_club"></select>Non se atopou ningún resultado.`
	if d, e = Parse(strings.NewReader(empty), query()); e != nil || len(d.Marks) != 0 {
		t.Fatal(d, e)
	}
}
func TestRequestFiltersCacheAndFailure(t *testing.T) {
	b, _ := os.ReadFile("testdata/results.html")
	calls := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		r.ParseForm()
		for k, v := range map[string]string{"combo_estament": "MASTE", "combo_split": "N", "combo_show_first": "100", "combo_club": "00123", "combo_chrono": "E"} {
			if r.Form.Get(k) != v {
				t.Errorf("%s=%s", k, r.Form.Get(k))
			}
		}
		if r.Form.Get("input_equate") != "" {
			t.Error("weighted conversion enabled")
		}
		w.Write(b)
	}))
	defer server.Close()
	c := New()
	c.URL = server.URL
	c.HTTP = server.Client()
	p, e := c.Preview(context.Background(), query())
	if e != nil || len(p.Data.Marks) != 1 {
		t.Fatal(p, e)
	}
	p, e = c.Preview(context.Background(), query())
	if e != nil || !p.Cached || calls.Load() != 1 {
		t.Fatal("cache miss", e)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer bad.Close()
	c = New()
	c.URL = bad.URL
	c.HTTP = bad.Client()
	if _, e = c.Preview(context.Background(), query()); e == nil {
		t.Fatal("503 accepted")
	}
}
func TestSplitDatesInsteadOfTruncating(t *testing.T) {
	b, _ := os.ReadFile("testdata/results.html")
	text := string(b)
	start := strings.Index(text, "<tr><td>")
	end := strings.Index(text[start:], "</tr>") + start + len("</tr>")
	row := text[start:end]
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		calls++
		response := text
		if r.Form.Get("input_from_date") == "2026-06-12" && r.Form.Get("input_to_date") == "2026-06-13" {
			rows := ""
			for i := 0; i < 100; i++ {
				rows += strings.Replace(row, "00001", fmt.Sprintf("%05d", i+1), 1)
			}
			response = strings.Replace(text, row, rows, 1)
		} else if r.Form.Get("input_from_date") == "2026-06-12" {
			response = `<select name="combo_club"></select>Non se atopou ningún resultado.`
		}
		fmt.Fprint(w, response)
	}))
	defer server.Close()
	c := New()
	c.URL = server.URL
	c.HTTP = server.Client()
	q := query()
	q.From = "2026-06-12"
	q.To = "2026-06-13"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, e := c.Preview(ctx, q)
	if e != nil || calls != 3 || len(p.Data.Marks) != 1 {
		t.Fatal(calls, p, e)
	}
}

func TestTopTenFallsBackToFilteredHistory(t *testing.T) {
	fixture, _ := os.ReadFile("testdata/results.html")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		r.ParseForm()
		b := fixture
		if r.Form.Get("input_best_mark") == "on" {
			b = []byte(strings.ReplaceAll(string(fixture), "E25", "E50"))
		}
		w.Write(b)
	}))
	defer server.Close()
	client := New()
	client.HTTP = server.Client()
	client.URL = server.URL
	q := query()
	q.TopTen = true
	p, e := client.Preview(context.Background(), q)
	if e != nil || calls != 2 || len(p.Data.Marks) != 1 || !p.Query.TopTen {
		t.Fatal(p, calls, e)
	}
}

func TestTenYearLicenseHistoryIncludesOldClub(t *testing.T) {
	fixture, err := os.ReadFile("testdata/results.html")
	if err != nil {
		t.Fatal(err)
	}
	fixture = []byte(strings.ReplaceAll(string(fixture), "20260613", "20180613"))
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		for key, want := range map[string]string{"combo_club": "", "input_license": "00001", "input_from_date": "2016-10-27", "input_to_date": "2026-10-27", "combo_chrono": "E", "combo_pool": "25"} {
			if r.Form.Get(key) != want {
				t.Errorf("wrong history filter %s", key)
			}
		}
		w.Write(fixture)
	}))
	defer remote.Close()
	c := New()
	c.URL = remote.URL
	c.HTTP = remote.Client()
	q := query()
	q.ClubID = ""
	q.License = "00001"
	q.From = "2016-10-27"
	q.To = "2026-10-27"
	q.Member = false
	p, err := c.Preview(context.Background(), q)
	if err != nil || len(p.Data.Marks) != 1 || p.Data.Marks[0].Date != "2018-06-13" || p.Data.Marks[0].ClubID != "00123" || p.Data.Marks[0].License != "00001" || p.Data.Swimmers[0].Member {
		t.Fatal("history unavailable or identity changed", err)
	}
	q.Event = "all"
	q.Pool = 0
	q.Chrono = ""
	queries, err := ExpandQuery(q)
	if err != nil || len(queries) != 4 {
		t.Fatal(queries, err)
	}
	for _, q := range queries {
		if q.ClubID != "" || q.License != "00001" || q.From != "2016-10-27" || q.To != "2026-10-27" || q.Member {
			t.Fatal("lost history scope")
		}
	}
}
