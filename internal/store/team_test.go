package store

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/domain"
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
