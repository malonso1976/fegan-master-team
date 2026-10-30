package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/malonso1976/fegan-master-team/internal/domain"
)

func TestImportEditBackupAndIdentity(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	d := domain.Data{Clubs: []domain.Club{{ID: "001", Name: "Club"}}, Swimmers: []domain.Swimmer{{License: "00001", Name: "Nadador", BirthYear: 1986, Gender: "M", ClubID: "001", Member: true, Available: true}}, Marks: []domain.Mark{{License: "00001", ClubID: "001", Event: "50L", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 3000, Source: "fegan", Gender: "M", Verified: true}}}
	r, e := s.Import(ctx, d)
	if e != nil || r.Inserted != 1 {
		t.Fatal(r, e)
	}
	r, e = s.Import(ctx, d)
	if e != nil || r.Duplicates != 1 {
		t.Fatal(r, e)
	}
	state, e := s.Snapshot(ctx)
	if e != nil {
		t.Fatal(e)
	}
	edited := state.Marks[0]
	edited.Centiseconds = 2900
	edited.Notes = "Corrección"
	if e = s.SaveMark(ctx, edited); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Import(ctx, d); e != nil {
		t.Fatal(e)
	}
	state, e = s.Snapshot(ctx)
	if e != nil || len(state.Marks) != 1 || state.Marks[0].Centiseconds != 2900 || state.Swimmers[0].License != "00001" {
		t.Fatal(state, e)
	}
	d.Swimmers[0].BirthYear = 1987
	d.Clubs = append(d.Clubs, domain.Club{ID: "002", Name: "Debe revertirse"})
	if _, e = s.Import(ctx, d); e == nil {
		t.Fatal("identity conflict accepted")
	}
	state, _ = s.Snapshot(ctx)
	if len(state.Clubs) != 1 {
		t.Fatal("partial import committed")
	}
	backup, e := s.Backup(ctx)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "restored.sqlite")
	if e = os.WriteFile(path, backup, 0600); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	copy, e := restored.Snapshot(ctx)
	if e != nil || copy.Marks[0].Notes != "Corrección" {
		t.Fatal(copy, e)
	}
	h, e := restored.History(ctx)
	if e != nil || len(h) != 1 {
		t.Fatal(h, e)
	}
}
func TestManualRelayWithoutRoster(t *testing.T) {
	ctx := context.Background()
	s, e := Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.SaveClub(ctx, domain.Club{ID: "001", Name: "Rival"}); e != nil {
		t.Fatal(e)
	}
	m := domain.Mark{ClubID: "001", Event: "4x50S", Pool: 25, Chrono: "E", Date: "2026-01-01", Centiseconds: 14000, Source: "manual", Gender: "X", AgeTotal: 160, RelayCategory: 160}
	if e = s.SaveMark(ctx, m); e != nil {
		t.Fatal(e)
	}
	m.License = "missing"
	m.Event = "50L"
	if e = s.SaveMark(ctx, m); e == nil {
		t.Fatal("unknown swimmer accepted")
	}
}

func TestSettingsMigrationBacksUpPreviousSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "team.sqlite")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveClub(ctx, domain.Club{ID: "00123", Name: "Existing"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DROP TABLE settings;UPDATE schema_version SET version=1"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	backups, _ := filepath.Glob(path + ".v1-*.bak")
	if len(backups) != 1 {
		t.Fatal(backups)
	}
	var version int
	s.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if version != 3 {
		t.Fatal(version)
	}
	if e = s.SetSetting(ctx, "home_club", "00123"); e != nil {
		t.Fatal(e)
	}
	var club string
	if e = s.Setting(ctx, "home_club", &club); e != nil || club != "00123" {
		t.Fatal(club, e)
	}
	d, e := s.Snapshot(ctx)
	if e != nil || len(d.Clubs) != 1 {
		t.Fatal(d, e)
	}
}
