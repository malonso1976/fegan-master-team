package web

import (
	"bytes"
	"context"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"github.com/marianoao/fegan-master-team/internal/store"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDeactivateSwimmerSelectionAPI(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.SaveClub(ctx, domain.Club{ID: "001", Name: "Club"}); err != nil {
		t.Fatal(err)
	}
	for _, license := range []string{"00001", "00002"} {
		if err = db.SaveSwimmer(ctx, domain.Swimmer{License: license, Name: "Prueba", BirthYear: 1980, Gender: "M", ClubID: "001", Member: true, Available: true}); err != nil {
			t.Fatal(err)
		}
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	call := func(body, token string) int {
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/swimmers/deactivate", bytes.NewBufferString(body))
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if call(`{"licenses":["00001"]}`, "") != 403 {
		t.Fatal("missing token accepted")
	}
	if call(`{"licenses":["00001","missing"]}`, s.token) != 400 {
		t.Fatal("missing row accepted")
	}
	if call(`{"licenses":[]}`, s.token) != 400 {
		t.Fatal("empty selection accepted")
	}
	state, _ := db.Snapshot(ctx)
	for _, swimmer := range state.Swimmers {
		if !swimmer.Member || !swimmer.Available {
			t.Fatal("failed request altered flags")
		}
	}
	if call(`{"licenses":["00001"]}`, s.token) != 200 {
		t.Fatal("deactivation failed")
	}
	state, _ = db.Snapshot(ctx)
	for _, swimmer := range state.Swimmers {
		want := swimmer.License == "00002"
		if swimmer.Member != want || swimmer.Available != want {
			t.Fatal("wrong selection updated")
		}
	}
}

func TestActivateSwimmerSelectionAPI(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.SaveClub(ctx, domain.Club{ID: "001", Name: "Club"}); err != nil {
		t.Fatal(err)
	}
	for _, license := range []string{"00001", "00002"} {
		if err = db.SaveSwimmer(ctx, domain.Swimmer{License: license, Name: "Prueba", BirthYear: 1980, Gender: "M", ClubID: "001"}); err != nil {
			t.Fatal(err)
		}
	}
	s := New(db, "127.0.0.1:12345")
	defer s.Close()
	call := func(body, token string) int {
		r := httptest.NewRequest("POST", "http://"+s.Host+"/api/swimmers/activate", bytes.NewBufferString(body))
		r.Header.Set("X-CSRF-Token", token)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if call(`{"licenses":["00001"]}`, "") != 403 {
		t.Fatal("token guard missing")
	}
	for _, body := range []string{`{"licenses":["00001","missing"]}`, `{"licenses":[]}`, `{"licenses":["00001"],"club_id":"other"}`} {
		if call(body, s.token) != 400 {
			t.Fatal("invalid activation accepted")
		}
	}
	state, _ := db.Snapshot(ctx)
	for _, swimmer := range state.Swimmers {
		if swimmer.Member || swimmer.Available {
			t.Fatal("failed activation changed data")
		}
	}
	if call(`{"licenses":["00001"]}`, s.token) != 200 {
		t.Fatal("activation failed")
	}
	state, _ = db.Snapshot(ctx)
	for _, swimmer := range state.Swimmers {
		want := swimmer.License == "00001"
		if swimmer.Member != want || swimmer.Available != want || swimmer.ClubID != "001" {
			t.Fatal("wrong activation target")
		}
	}
}
