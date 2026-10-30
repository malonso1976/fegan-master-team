package store

import (
	"context"
	"github.com/malonso1976/fegan-master-team/internal/domain"
	"path/filepath"
	"testing"
)

func TestDeactivateSelectionPreservesProfilesMarksAndImportDecisions(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	club := domain.Club{ID: "001", Name: "Club de prueba"}
	if err = db.SaveClub(ctx, club); err != nil {
		t.Fatal(err)
	}
	data := domain.Data{Clubs: []domain.Club{club}}
	for _, license := range []string{"00001", "00002", "00003"} {
		s := domain.Swimmer{License: license, Name: "Prueba", BirthYear: 1980, Gender: "M", ClubID: club.ID, Member: true, Available: true}
		if err = db.SaveSwimmer(ctx, s); err != nil {
			t.Fatal(err)
		}
		m := domain.Mark{License: license, ClubID: club.ID, Event: "50L", Gender: "M", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "fegan"}
		if err = db.SaveMark(ctx, m); err != nil {
			t.Fatal(err)
		}
		data.Swimmers = append(data.Swimmers, s)
		data.Marks = append(data.Marks, m)
	}
	updated, err := db.DeactivateSwimmers(ctx, []string{"00001", "00002", "00001"})
	if err != nil || updated != 2 {
		t.Fatal(updated, err)
	}
	history, err := db.History(ctx)
	if err != nil || len(history) != 2 {
		t.Fatal("one history entry per changed profile", err)
	}
	if updated, err = db.DeactivateSwimmers(ctx, []string{"00001", "00002"}); err != nil || updated != 0 {
		t.Fatal(updated, err)
	}
	if _, err = db.Import(ctx, data); err != nil {
		t.Fatal(err)
	}
	snapshot, err := db.Snapshot(ctx)
	if err != nil || len(snapshot.Marks) != 3 || len(snapshot.Swimmers) != 3 {
		t.Fatal("records lost", err)
	}
	for _, s := range snapshot.Swimmers {
		want := s.License == "00003"
		if s.Member != want || s.Available != want || s.ClubID != club.ID || s.BirthYear != 1980 || s.Name != "Prueba" {
			t.Fatal("profile altered or reactivated")
		}
	}
	// A missing row after a valid selection must roll back every update and its history.
	if _, err = db.DeactivateSwimmers(ctx, []string{"00003", "missing"}); err == nil {
		t.Fatal("missing license accepted")
	}
	snapshot, err = db.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range snapshot.Swimmers {
		if s.License == "00003" && (!s.Member || !s.Available) {
			t.Fatal("partial update committed")
		}
	}
	for _, selection := range [][]string{nil, {"bad license"}, make([]string, 1001)} {
		if _, err = db.DeactivateSwimmers(ctx, selection); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
}

func TestActivateSelectionIsAtomicAndPreservesHistoricalData(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	club := domain.Club{ID: "001", Name: "Club de prueba"}
	if err = db.SaveClub(ctx, club); err != nil {
		t.Fatal(err)
	}
	original := domain.Data{Clubs: []domain.Club{club}}
	for _, license := range []string{"00001", "00002", "00003"} {
		swimmer := domain.Swimmer{License: license, Name: "Prueba", BirthYear: 1980, Gender: "M", ClubID: club.ID}
		if err = db.SaveSwimmer(ctx, swimmer); err != nil {
			t.Fatal(err)
		}
		original.Swimmers = append(original.Swimmers, swimmer)
		mark := domain.Mark{License: license, ClubID: club.ID, Event: "50L", Gender: "M", Pool: 25, Chrono: "E", Date: "2023-01-01", Centiseconds: 3000, Source: "fegan"}
		if err = db.SaveMark(ctx, mark); err != nil {
			t.Fatal(err)
		}
		original.Marks = append(original.Marks, mark)
	}
	changed, err := db.ActivateSwimmers(ctx, []string{"00001", "00002", "00001"})
	if err != nil || changed != 2 {
		t.Fatal(changed, err)
	}
	history, err := db.History(ctx)
	if err != nil || len(history) != 2 {
		t.Fatal("activation history missing", err)
	}
	if changed, err = db.ActivateSwimmers(ctx, []string{"00001", "00002"}); err != nil || changed != 0 {
		t.Fatal("activation not idempotent", err)
	}
	if _, err = db.Import(ctx, original); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ActivateSwimmers(ctx, []string{"00003", "missing"}); err == nil {
		t.Fatal("invalid selection accepted")
	}
	state, err := db.Snapshot(ctx)
	if err != nil || len(state.Marks) != 3 || len(state.Swimmers) != 3 {
		t.Fatal("data lost", err)
	}
	for _, swimmer := range state.Swimmers {
		want := swimmer.License != "00003"
		if swimmer.Member != want || swimmer.Available != want || swimmer.ClubID != club.ID || swimmer.Name != "Prueba" || swimmer.BirthYear != 1980 {
			t.Fatal("activation changed wrong fields, was reverted by import or committed partially")
		}
	}
	for _, selection := range [][]string{nil, {"bad license"}, make([]string, 1001)} {
		if _, err = db.ActivateSwimmers(ctx, selection); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
}
