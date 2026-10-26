package store

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"path/filepath"
	"testing"
)

func resetFixture() domain.Data {
	return domain.Data{Clubs: []domain.Club{{ID: "001", Name: "Club"}}, Swimmers: []domain.Swimmer{{License: "01", Name: "Prueba", BirthYear: 1980, Gender: "M", ClubID: "001"}}, Marks: []domain.Mark{{License: "01", ClubID: "001", Event: "50L", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "fegan", Gender: "M"}}}
}
func TestResetClearsAllUserStateAndAllowsFreshImport(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	d := resetFixture()
	if _, e = s.Import(ctx, d, map[string]string{"test": "seed"}); e != nil {
		t.Fatal(e)
	}
	if e = s.SetSetting(ctx, "home_club", "001"); e != nil {
		t.Fatal(e)
	}
	state, _ := s.Snapshot(ctx)
	if _, e = s.Delete(ctx, "marks", state.Marks[0].ID); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveClub(ctx, d.Clubs[0]); e != nil {
		t.Fatal(e)
	} // creates an old version
	if e = s.Reset(ctx); e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"clubs", "swimmers", "marks", "history", "settings", "suppressed"} {
		var count int
		if e = s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); e != nil || count != 0 {
			t.Fatal(table, count, e)
		}
	}
	var version int
	s.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if version != 3 {
		t.Fatal(version)
	}
	result, e := s.Import(ctx, d)
	if e != nil || result.Inserted != 1 {
		t.Fatal("old exclusion survived", result, e)
	}
}
func TestResetRollsBackWholeDatabaseOnFailure(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.Import(ctx, resetFixture()); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("CREATE TRIGGER prevent_reset BEFORE DELETE ON clubs BEGIN SELECT RAISE(ABORT,'test failure'); END"); e != nil {
		t.Fatal(e)
	}
	if e = s.Reset(ctx); e == nil {
		t.Fatal("reset should fail")
	}
	state, e := s.Snapshot(ctx)
	if e != nil || len(state.Clubs) != 1 || len(state.Swimmers) != 1 || len(state.Marks) != 1 {
		t.Fatal("partial reset", state, e)
	}
}
