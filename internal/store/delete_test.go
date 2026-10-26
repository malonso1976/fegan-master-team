package store

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"path/filepath"
	"testing"
)

func TestCascadeDeletionAndReimportSuppression(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	d := domain.Data{Clubs: []domain.Club{{ID: "001", Name: "Own"}, {ID: "002", Name: "Old"}}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}
	for _, l := range []string{"a", "b", "c", "d"} {
		d.Swimmers = append(d.Swimmers, domain.Swimmer{License: l, Name: l, BirthYear: 1980, Gender: "M", ClubID: "001", Member: true, Available: true})
	}
	for _, l := range []string{"a", "b"} {
		d.Marks = append(d.Marks, domain.Mark{License: l, ClubID: "002", Event: "50L", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "fegan", Gender: "M"})
	}
	d.Marks = append(d.Marks, domain.Mark{ClubID: "001", Event: "4x50L", RelayCategory: 120, RelayLicenses: []string{"a", "b", "c", "d"}, Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 12000, Source: "fegan-pdf", Gender: "M"})
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, _ := s.Snapshot(ctx)
	m := state.Marks[0]
	for _, mark := range state.Marks {
		if mark.License == "a" {
			m = mark
			break
		}
	}
	m.Centiseconds--
	if e = s.SaveMark(ctx, m); e != nil {
		t.Fatal(e)
	}
	out, e := s.Delete(ctx, "swimmers", "a")
	if e != nil || out.Swimmers != 1 || out.Marks != 2 {
		t.Fatal(out, e)
	}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, _ = s.Snapshot(ctx)
	if len(state.Swimmers) != 3 || len(state.Marks) != 1 {
		t.Fatal(state)
	}
	h, _ := s.History(ctx)
	if len(h) != 0 {
		t.Fatal("deleted versions remain", h)
	}
	if e = s.SetSetting(ctx, "home_club", "001"); e != nil {
		t.Fatal(e)
	}
	out, e = s.Delete(ctx, "clubs", "001")
	if e != nil || out.Swimmers != 3 || out.Marks != 1 || !out.HomeCleared {
		t.Fatal(out, e)
	}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, _ = s.Snapshot(ctx)
	if len(state.Swimmers) != 0 || len(state.Marks) != 0 || len(state.Clubs) != 1 {
		t.Fatal(state)
	}
	var home string
	s.Setting(ctx, "home_club", &home)
	if home != "" {
		t.Fatal(home)
	}
	if e = s.SaveClub(ctx, d.Clubs[0]); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveSwimmer(ctx, d.Swimmers[0]); e != nil {
		t.Fatal(e)
	}
	state, _ = s.Snapshot(ctx)
	if len(state.Swimmers) != 1 {
		t.Fatal("manual recreation unavailable")
	}
}
func TestDeletedEditedMarkCannotReturnWithOriginalFingerprint(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	d := domain.Data{Clubs: []domain.Club{{ID: "001", Name: "Club"}}, Swimmers: []domain.Swimmer{{License: "01", Name: "A", BirthYear: 1980, Gender: "M", ClubID: "001"}}, Marks: []domain.Mark{{License: "01", ClubID: "001", Event: "50L", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "fegan", Gender: "M"}}}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, _ := s.Snapshot(ctx)
	m := state.Marks[0]
	m.Centiseconds = 2900
	if e = s.SaveMark(ctx, m); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Delete(ctx, "marks", m.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	d.Marks[0].Centiseconds = 2900
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, _ = s.Snapshot(ctx)
	if len(state.Marks) != 0 || len(state.Swimmers) != 1 {
		t.Fatal(state)
	}
}

func TestSchemaTwoMigrationBacksUpBeforeExclusions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "team.sqlite")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveClub(context.Background(), domain.Club{ID: "001", Name: "Preserved"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DROP TABLE suppressed;UPDATE schema_version SET version=2"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	backups, _ := filepath.Glob(path + ".v2-*.bak")
	if len(backups) != 1 {
		t.Fatal(backups)
	}
	var version int
	if e = s.db.QueryRow("SELECT version FROM schema_version").Scan(&version); e != nil || version != 3 {
		t.Fatal(version, e)
	}
	d, e := s.Snapshot(context.Background())
	if e != nil || len(d.Clubs) != 1 {
		t.Fatal(d, e)
	}
	if _, e = s.Delete(context.Background(), "clubs", "001"); e != nil {
		t.Fatal(e)
	}
}

func TestDeleteHistoricalClubKeepsSwimmerCurrentClub(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	d := domain.Data{Clubs: []domain.Club{{ID: "001", Name: "Current"}, {ID: "002", Name: "Historical"}}, Swimmers: []domain.Swimmer{{License: "a", Name: "A", ClubID: "001", Gender: "M", BirthYear: 1980}}, Marks: []domain.Mark{{License: "a", ClubID: "002", Event: "50L", Date: "2026-01-01", Pool: 25, Chrono: "E", Gender: "M", Centiseconds: 3000, Source: "fegan"}}}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	result, e := s.Delete(ctx, "clubs", "002")
	if e != nil || result.Swimmers != 0 || result.Marks != 1 {
		t.Fatal(result, e)
	}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, e := s.Snapshot(ctx)
	if e != nil || len(state.Swimmers) != 1 || len(state.Marks) != 0 || len(state.Clubs) != 1 {
		t.Fatal(state, e)
	}
}
