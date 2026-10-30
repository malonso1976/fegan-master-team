package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/store"
)

func TestLocalGuardAndPersistence(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(db, "127.0.0.1:12345")
	h := s.Handler()
	call := func(method, path, host, origin, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://"+host+path, bytes.NewBufferString(body))
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if token != "" {
			r.Header.Set("X-CSRF-Token", token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, test := range []struct{ method, path, host, origin, token string }{
		{"GET", "/api/data", "evil.example:12345", "", ""}, {"GET", "/api/session", s.Host, "https://evil.example", ""}, {"POST", "/api/clubs", s.Host, "", ""},
	} {
		w := call(test.method, test.path, test.host, test.origin, test.token, `{"id":"001","name":"Club"}`)
		if w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := call("GET", "/api/session", s.Host, "", "", "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var session map[string]any
	json.Unmarshal(w.Body.Bytes(), &session)
	token := session["token"].(string)
	w = call("POST", "/api/clubs", s.Host, "http://"+s.Host, token, `{"id":"001","name":"Club"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	d, e := db.Snapshot(context.Background())
	if e != nil || len(d.Clubs) != 1 {
		t.Fatal(d, e)
	}
	w = call("POST", "/api/clubs", s.Host, "", token, `{"id":"002","name":"Club"} {}`)
	if w.Code != 400 {
		t.Fatal("trailing json accepted")
	}
	w = call("GET", "/", s.Host, "", "", "")
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("MASTER TEAM")) {
		t.Fatal("UI missing")
	}
	// A failed import must not persist a valid first row.
	invalid := domain.Data{Clubs: []domain.Club{{ID: "003", Name: "Temporary"}}, Swimmers: []domain.Swimmer{{License: "x", ClubID: "missing", Name: "Test", BirthYear: 1986, Gender: "M"}}}
	if _, e = db.Import(context.Background(), invalid); e == nil {
		t.Fatal("invalid import accepted")
	}
	d, _ = db.Snapshot(context.Background())
	if len(d.Clubs) != 1 {
		t.Fatal("partial commit")
	}
}

func TestSeasonRecommendationAndCategoryAPI(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ctx := context.Background()
	if e = db.SaveClub(ctx, domain.Club{ID: "001", Name: "Club"}); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 4; i++ {
		license := string(rune('a' + i))
		if e = db.SaveSwimmer(ctx, domain.Swimmer{License: license, Name: license, BirthYear: 1982, Gender: "M", ClubID: "001", Member: true, Available: true}); e != nil {
			t.Fatal(e)
		}
		if e = db.SaveMark(ctx, domain.Mark{License: license, ClubID: "001", Event: "50L", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "manual", Gender: "M", Verified: true}); e != nil {
			t.Fatal(e)
		}
	}
	s := New(db, "127.0.0.1:12345")
	h := s.Handler()
	call := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://"+s.Host+path, bytes.NewBufferString(body))
		r.Header.Set("X-CSRF-Token", s.token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := call("/api/recommend", `{"license":"a","season_start":2026,"from":"2025-01-01","to":"2026-12-31","pool":25,"chrono":"E","verified_only":true}`)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("45–49")) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = call("/api/relays", `{"club_id":"001","event":"4x50L","gender":"M","season_start":2026,"from":"2025-01-01","to":"2026-12-31","pool":25,"chrono":"E","verified_only":true,"category":0,"competition":"championship","limit":2}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var advice domain.RelayAdvice
	if e = json.Unmarshal(w.Body.Bytes(), &advice); e != nil || len(advice.Groups) != 8 || len(advice.Groups[3].Results) != 1 || advice.Groups[3].Results[0].AgeTotal != 180 {
		t.Fatal(advice, e)
	}
}

func TestRelayAPIReturnsRivalsWithoutOwnComposition(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	for _, id := range []string{"001", "002"} {
		if err = db.SaveClub(ctx, domain.Club{ID: id, Name: "Club " + id}); err != nil {
			t.Fatal(err)
		}
	}
	if err = db.SaveMark(ctx, domain.Mark{ClubID: "002", Event: "4x50L", Gender: "F", RelayCategory: 120, Pool: 25, Chrono: "E", Date: "2026-06-13", Centiseconds: 13000, Source: "manual", Verified: true}); err != nil {
		t.Fatal(err)
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	r := httptest.NewRequest("POST", "http://"+s.Host+"/api/relays", bytes.NewBufferString(`{"club_id":"001","event":"4x50L","gender":"F","season_start":2026,"from":"2024-10-27","to":"2026-10-27","verified_only":true,"category":120,"limit":2}`))
	r.Header.Set("X-CSRF-Token", s.token)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var groups []relayComparison
	if err = json.Unmarshal(w.Body.Bytes(), &groups); err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatal(groups)
	}
	for _, group := range groups {
		g := group.Advice.Groups[0]
		want := 0
		if group.Pool == 25 && group.Chrono == "" {
			want = 1
		}
		if len(g.Results) != 0 || len(g.Rivals) != want || len(g.Coverage) != 1 || g.Coverage[0].Swimmers != 0 {
			t.Fatalf("wrong partition: %+v", group)
		}
		if want == 1 && (g.Rivals[0].Club.ID != "002" || g.Rivals[0].Mark.Centiseconds != 13000) {
			t.Fatal(g.Rivals)
		}
	}
}
