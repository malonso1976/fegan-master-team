package web

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/fegan"
	"github.com/malonso1976/fegan-master-team/internal/store"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalPDFPreviewReviewConfirmAndDuplicate(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	h := s.Handler()
	ctx := context.Background()
	b, _ := os.ReadFile("../fegan/testdata/catalog.pdf")
	catalog, e := fegan.ParseClubsPDF(ctx, b, "")
	if e != nil {
		t.Fatal(e)
	}
	db.SetSetting(ctx, "club_catalog", catalog)
	pdf, _ := os.ReadFile("../fegan/testdata/relays.pdf")
	upload := func(content []byte, aliases string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, _ := form.CreateFormFile("file", "results.pdf")
		part.Write(content)
		form.WriteField("date", "2025-01-12")
		form.WriteField("championship", "winter")
		form.WriteField("chrono", "E")
		form.WriteField("aliases", aliases)
		form.Close()
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/relay-pdf/preview", &body)
		r.Header.Set("Content-Type", form.FormDataContentType())
		r.Header.Set("X-CSRF-Token", s.token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := upload([]byte("not PDF"), "{}"); w.Code != 422 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := upload(pdf, `{"UNKNOWN":"missing"}`); w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
	for i := 0; i < 2; i++ {
		w := upload(pdf, "{}")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var result struct {
			ID      string
			Preview fegan.Preview
		}
		if e = json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if len(result.Preview.Data.Marks) != 1 || len(result.Preview.Warnings) < 2 {
			t.Fatal(result)
		}
		m := result.Preview.Data.Marks[0]
		if m.Source != "local-pdf" || m.Verified || !strings.HasPrefix(m.SourceURL, "local-pdf:") {
			t.Fatal(m)
		}
		state, _ := db.Snapshot(ctx)
		if i == 0 && len(state.Marks) != 0 {
			t.Fatal("preview saved data")
		}
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/import/confirm", strings.NewReader(`{"id":"`+result.ID+`"}`))
		r.Header.Set("X-CSRF-Token", s.token)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	state, _ := db.Snapshot(ctx)
	if len(state.Marks) != 1 {
		t.Fatal(state)
	}
}
func TestAutomaticComparisonPartitionsAndRivalDetails(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ctx := context.Background()
	db.SaveClub(ctx, domain.Club{ID: "001", Name: "Club"})
	for _, license := range []string{"a", "b"} {
		db.SaveSwimmer(ctx, domain.Swimmer{License: license, Name: license, BirthYear: 1980, Gender: "M", ClubID: "001"})
		for _, pool := range []int{25, 50} {
			for _, chrono := range []string{"E", "M"} {
				m := domain.Mark{License: license, ClubID: "001", Event: "50L", Pool: pool, Chrono: chrono, Date: "2026-01-01", Centiseconds: 3000, Source: "manual", Gender: "M"}
				if license == "b" {
					m.Centiseconds = 2900
				}
				if e = db.SaveMark(ctx, m); e != nil {
					t.Fatal(e)
				}
			}
		}
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	r := httptest.NewRequest("POST", "http://"+s.Host+"/api/recommend", strings.NewReader(`{"license":"a","pool":25,"from":"2025-01-01","to":"2026-10-26"}`))
	r.Header.Set("X-CSRF-Token", s.token)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var rows []domain.Recommendation
	json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Fatal(rows)
	}
	for _, row := range rows {
		if row.Pool != 25 || len(row.Details) != 1 || row.Details[0].Swimmer.License != "b" || row.Details[0].Mark.Centiseconds != 2900 || row.OwnMark.Centiseconds != 3000 {
			t.Fatal(row)
		}
	}
}

func TestWideImportPreparesAsynchronouslyWithoutSaving(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<select name="combo_club"></select>Non se atopou ningún resultado.`))
	}))
	defer remote.Close()
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	s.Fegan.URL = remote.URL
	s.Fegan.HTTP = remote.Client()
	h := s.Handler()
	r := httptest.NewRequest("POST", "http://"+s.Host+"/api/import/start", strings.NewReader(`{"club_id":"00123","event":"50L","from":"2025-01-01","to":"2026-10-26"}`))
	r.Header.Set("X-CSRF-Token", s.token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var job importJob
	json.Unmarshal(w.Body.Bytes(), &job)
	if job.Ready || job.ID == "" {
		t.Fatal(job)
	}
	id := job.ID
	deadline := time.Now().Add(15 * time.Second)
	for !job.Ready && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://"+s.Host+"/api/import/status/"+id, nil))
		if e = json.Unmarshal(w.Body.Bytes(), &job); e != nil {
			t.Fatal(e)
		}
	}
	if !job.Ready || job.Error != "" || job.Preview == nil || job.Preview.Requests != 8 {
		t.Fatal(job)
	}
	d, e := db.Snapshot(context.Background())
	if e != nil || len(d.Clubs) != 0 || len(d.Marks) != 0 {
		t.Fatal(d, e)
	}
	s.mu.Lock()
	_, exists := s.previews[job.ID]
	s.mu.Unlock()
	if !exists {
		t.Fatal("preview not retained for review")
	}
}

func TestOpportunityAPIAlwaysUsesRecentRivalsAndHistoricalOwnTiming(t *testing.T) {
	ctx := context.Background()
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	now := time.Now()
	if e = db.SaveClub(ctx, domain.Club{ID: "001", Name: "Prueba"}); e != nil {
		t.Fatal(e)
	}
	for _, l := range []string{"a", "b"} {
		if e = db.SaveSwimmer(ctx, domain.Swimmer{License: l, Name: l, BirthYear: now.Year() - 40, Gender: "M", ClubID: "001"}); e != nil {
			t.Fatal(e)
		}
	}
	marks := []struct {
		l, chrono string
		years, cs int
	}{{"a", "M", -5, 2600}, {"a", "E", -1, 2800}, {"b", "M", -3, 2400}, {"b", "M", -1, 2700}, {"b", "E", -1, 2000}}
	for _, x := range marks {
		if e = db.SaveMark(ctx, domain.Mark{License: x.l, ClubID: "001", Event: "50L", Pool: 25, Chrono: x.chrono, Date: now.AddDate(x.years, 0, 0).Format("2006-01-02"), Centiseconds: x.cs, Source: "manual", Gender: "M", Verified: true}); e != nil {
			t.Fatal(e)
		}
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	r := httptest.NewRequest("POST", "http://"+s.Host+"/api/recommend", strings.NewReader(`{"license":"a","pool":25,"chrono":"E","from":"1900-01-01","to":"2199-01-01"}`))
	r.Header.Set("X-CSRF-Token", s.token)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	var rows []domain.Recommendation
	if e = json.Unmarshal(w.Body.Bytes(), &rows); e != nil || w.Code != 200 || len(rows) != 1 {
		t.Fatal(w.Code, w.Body.String(), e)
	}
	row := rows[0]
	if row.Time != 2600 || row.Chrono != "M" || row.Rank != 1 || len(row.Details) != 1 || row.Details[0].Mark.Centiseconds != 2700 {
		t.Fatal(row)
	}
	r = httptest.NewRequest("POST", "http://"+s.Host+"/api/recommend", strings.NewReader(`{"license":"a"}`))
	r.Header.Set("X-CSRF-Token", s.token)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if e = json.Unmarshal(w.Body.Bytes(), &rows); e != nil || w.Code != 200 || len(rows) != 1 {
		t.Fatal(w.Code, w.Body.String(), e)
	}
	row = rows[0]
	if row.Pool != 50 || row.Time != 2689 || row.OwnMark.Centiseconds != 2600 || row.OwnMark.Pool != 25 || row.Chrono != "E" || row.Conversion == nil || row.Rank != 2 || len(row.Details) != 1 || row.Details[0].Time != 2060 {
		t.Fatal(row)
	}
}
