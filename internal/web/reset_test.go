package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/fegan"
	"github.com/malonso1976/fegan-master-team/internal/store"
)

func TestResetRequiresConfirmationInvalidatesOldTabsAndPreviews(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	ctx := context.Background()
	if e = db.SaveClub(ctx, domain.Club{ID: "001", Name: "Club"}); e != nil {
		t.Fatal(e)
	}
	if e = db.SetSetting(ctx, "home_club", "001"); e != nil {
		t.Fatal(e)
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	h := s.Handler()
	oldToken := s.token
	id := s.retain(fegan.Preview{Data: domain.Data{Clubs: []domain.Club{{ID: "002", Name: "Stale"}}}})
	call := func(path, token, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://"+s.Host+path, strings.NewReader(body))
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, c := range []struct {
		token, body string
		code        int
	}{{"", `{"confirmation":"REINICIAR"}`, 403}, {oldToken, `{}`, 400}, {oldToken, `{"confirmation":"reiniciar"}`, 400}, {oldToken, `{"confirmation":"REINICIAR","extra":true}`, 400}} {
		if w := call("/api/reset", c.token, c.body); w.Code != c.code {
			t.Fatal(w.Code, w.Body.String())
		}
		state, _ := db.Snapshot(ctx)
		if len(state.Clubs) != 1 || s.workCtx.Err() != nil {
			t.Fatal("unconfirmed reset changed state")
		}
	}
	if w := call("/api/reset", oldToken, `{"confirmation":"REINICIAR"}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("/api/clubs", oldToken, `{"id":"003","name":"Stale tab"}`); w.Code != 403 {
		t.Fatal("stale tab accepted", w.Code)
	}
	if w := call("/api/import/confirm", s.token, `{"id":"`+id+`"}`); w.Code != 400 {
		t.Fatal("old preview survived", w.Code)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://"+s.Host+"/api/setup", nil))
	var setup struct {
		Home    string `json:"home_club_id"`
		Catalog fegan.Catalog
		Sync    struct {
			Running bool
			ClubID  string `json:"club_id"`
		}
	}
	if e = json.Unmarshal(w.Body.Bytes(), &setup); e != nil || setup.Home != "" || len(setup.Catalog.Clubs) != 0 || setup.Sync.Running || setup.Sync.ClubID != "" {
		t.Fatal(setup, e)
	}
	state, _ := db.Snapshot(ctx)
	if len(state.Clubs) != 0 {
		t.Fatal(state)
	}
	if w := call("/api/clubs", s.token, `{"id":"001","name":"Fresh club"}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestResetCancelsActiveCatalogRequestAndBackgroundSync(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(started) }); <-r.Context().Done() }))
	defer source.Close()
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	s.Fegan.DocumentsBase = source.URL
	s.Fegan.URL = source.URL
	s.Fegan.HTTP = source.Client()
	h := s.Handler()
	token := s.token
	catalogDone := make(chan struct{})
	go func() {
		defer close(catalogDone)
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/catalog", strings.NewReader(`{}`))
		r.Header.Set("X-CSRF-Token", token)
		h.ServeHTTP(httptest.NewRecorder(), r)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("catalog request did not start")
	}
	// The worker is also active while the reset cancels HTTP work and waits for it.
	s.Sync.Start("001", "all", false)
	r := httptest.NewRequest("POST", "http://"+s.Host+"/api/import/start", strings.NewReader(`{"club_id":"001","event":"50L","from":"2025-01-01","to":"2026-01-01"}`))
	r.Header.Set("X-CSRF-Token", token)
	startedJob := httptest.NewRecorder()
	h.ServeHTTP(startedJob, r)
	if startedJob.Code != 200 {
		t.Fatal(startedJob.Code, startedJob.Body.String())
	}
	var job importJob
	json.Unmarshal(startedJob.Body.Bytes(), &job)

	resetDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/reset", strings.NewReader(`{"confirmation":"REINICIAR"}`))
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		resetDone <- w
	}()
	select {
	case w := <-resetDone:
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reset did not cancel pending work")
	}
	<-catalogDone
	result := httptest.NewRecorder()
	h.ServeHTTP(result, httptest.NewRequest("GET", "http://"+s.Host+"/api/import/status/"+job.ID, nil))
	if result.Code != 404 {
		t.Fatal("old async job survived reset", result.Code)
	}
	var catalog fegan.Catalog
	if e = db.Setting(context.Background(), "club_catalog", &catalog); e != nil || len(catalog.Clubs) != 0 {
		t.Fatal("old catalog survived", catalog, e)
	}
	state, e := db.Snapshot(context.Background())
	if e != nil || len(state.Clubs) != 0 || s.Sync.Status().Running {
		t.Fatal(state, e)
	}
}
