package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/malonso1976/fegan-master-team/internal/domain"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	existed := statErr == nil
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	schema := `PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL);
 INSERT INTO schema_version SELECT 3 WHERE NOT EXISTS(SELECT 1 FROM schema_version);
 CREATE TABLE IF NOT EXISTS clubs(id TEXT PRIMARY KEY, payload TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS swimmers(license TEXT PRIMARY KEY, club_id TEXT NOT NULL REFERENCES clubs(id), payload TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS marks(id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, club_id TEXT NOT NULL REFERENCES clubs(id), license TEXT REFERENCES swimmers(license), payload TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS history(id INTEGER PRIMARY KEY, entity TEXT NOT NULL, entity_id TEXT NOT NULL, before TEXT NOT NULL, at TEXT NOT NULL DEFAULT(strftime('%Y-%m-%dT%H:%M:%fZ','now')));
 CREATE INDEX IF NOT EXISTS swimmer_club ON swimmers(club_id);
 CREATE INDEX IF NOT EXISTS mark_license ON marks(license);`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	var version int
	if err = db.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil || (version != 1 && version != 2 && version != 3) {
		db.Close()
		return nil, errors.New("versión de base de datos incompatible")
	}
	if version < 3 && existed {
		backup := fmt.Sprintf("%s.v%d-%s.bak", path, version, uuid.NewString())
		if _, err = db.Exec("VACUUM INTO '" + strings.ReplaceAll(backup, "'", "''") + "'"); err != nil {
			db.Close()
			return nil, err
		}
		if err = os.Chmod(backup, 0600); err != nil {
			db.Close()
			return nil, err
		}
	}
	tx, err := db.Begin()
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = tx.Exec("CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL);CREATE TABLE IF NOT EXISTS suppressed(kind TEXT NOT NULL,key TEXT NOT NULL,PRIMARY KEY(kind,key));UPDATE schema_version SET version=3"); err != nil {
		tx.Rollback()
		db.Close()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		db.Close()
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Snapshot(ctx context.Context) (domain.Data, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot(ctx)
}
func (s *Store) snapshot(ctx context.Context) (domain.Data, error) {
	d := domain.Data{Clubs: []domain.Club{}, Swimmers: []domain.Swimmer{}, Marks: []domain.Mark{}}
	for _, item := range []struct {
		query string
		add   func([]byte) error
	}{
		{"SELECT payload FROM clubs ORDER BY id", func(b []byte) error {
			var v domain.Club
			if e := json.Unmarshal(b, &v); e != nil {
				return e
			}
			d.Clubs = append(d.Clubs, v)
			return nil
		}},
		{"SELECT payload FROM swimmers ORDER BY license", func(b []byte) error {
			var v domain.Swimmer
			if e := json.Unmarshal(b, &v); e != nil {
				return e
			}
			d.Swimmers = append(d.Swimmers, v)
			return nil
		}},
		{"SELECT payload FROM marks ORDER BY json_extract(payload,'$.date') DESC,id", func(b []byte) error {
			var v domain.Mark
			if e := json.Unmarshal(b, &v); e != nil {
				return e
			}
			d.Marks = append(d.Marks, v)
			return nil
		}},
	} {
		rows, e := s.db.QueryContext(ctx, item.query)
		if e != nil {
			return d, e
		}
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e == nil {
				e = item.add(b)
			}
			if e != nil {
				rows.Close()
				return d, e
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return d, e
		}
	}
	return d, nil
}
func payload(v any) string { b, _ := json.Marshal(v); return string(b) }
func saveClub(ctx context.Context, tx *sql.Tx, c domain.Club) error {
	if e := c.Validate(); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "INSERT INTO clubs(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", c.ID, payload(c))
	return e
}
func history(ctx context.Context, tx *sql.Tx, table, key, id string) error {
	_, e := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO history(entity,entity_id,before) SELECT ?,?,payload FROM %s WHERE %s=?", table, key), table, id, id)
	return e
}
func (s *Store) SaveClub(ctx context.Context, c domain.Club) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = history(ctx, tx, "clubs", "id", c.ID); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM suppressed WHERE kind='clubs' AND key=?", suppressionKey(c.ID)); e != nil {
		return e
	}
	if e = saveClub(ctx, tx, c); e != nil {
		return e
	}
	return tx.Commit()
}
func saveSwimmer(ctx context.Context, tx *sql.Tx, v domain.Swimmer) error {
	if e := v.Validate(); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "INSERT INTO swimmers(license,club_id,payload) VALUES(?,?,?) ON CONFLICT(license) DO UPDATE SET club_id=excluded.club_id,payload=excluded.payload", v.License, v.ClubID, payload(v))
	return e
}
func (s *Store) SaveSwimmer(ctx context.Context, v domain.Swimmer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = history(ctx, tx, "swimmers", "license", v.License); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM suppressed WHERE kind='swimmers' AND key=?", suppressionKey(v.License)); e != nil {
		return e
	}
	if e = saveSwimmer(ctx, tx, v); e != nil {
		return e
	}
	return tx.Commit()
}
func fingerprint(m domain.Mark) string {
	m.ID = ""
	m.Source = ""
	m.SourceURL = ""
	m.RetrievedAt = ""
	m.Verified = false
	m.Notes = ""
	sum := sha256.Sum256([]byte(payload(m)))
	return hex.EncodeToString(sum[:])
}
func validateRelations(ctx context.Context, tx *sql.Tx, m domain.Mark) error {
	if e := m.Validate(); e != nil {
		return e
	}
	licenses := []string{m.License}
	if domain.Relay(m.Event) {
		licenses = m.RelayLicenses
	}
	female := 0
	ageTotal := 0
	seasonDate, _ := time.Parse("2006-01-02", m.Date)
	for _, l := range licenses {
		var b string
		if e := tx.QueryRowContext(ctx, "SELECT payload FROM swimmers WHERE license=?", l).Scan(&b); e != nil {
			return errors.New("una licencia de la marca no existe")
		}
		var swimmer domain.Swimmer
		if e := json.Unmarshal([]byte(b), &swimmer); e != nil {
			return e
		}
		if domain.Relay(m.Event) {
			age := domain.SeasonAge(domain.SeasonStartOn(seasonDate), swimmer.BirthYear)
			ageTotal += age
			if age < 20 || (age < 25 && m.RelayCategory != 80) {
				return errors.New("integrante no elegible por edad/premáster para la categoría del relevo")
			}
		}
		if swimmer.Gender == "F" {
			female++
		}
		if !domain.Relay(m.Event) && swimmer.Gender != m.Gender {
			return errors.New("categoría de la marca no coincide con el nadador")
		}
	}
	if domain.Relay(m.Event) && len(m.RelayLicenses) > 0 && ageTotal < m.RelayCategory {
		return errors.New("suma de edades inferior a la categoría del relevo")
	}
	if domain.Relay(m.Event) && len(m.RelayLicenses) > 0 && m.AgeTotal != 0 && m.AgeTotal != ageTotal {
		return errors.New("suma de edades no coincide con los integrantes en la temporada del resultado")
	}
	if domain.Relay(m.Event) && len(m.RelayLicenses) > 0 && ((m.Gender == "M" && female != 0) || (m.Gender == "F" && female != 4) || (m.Gender == "X" && female != 2)) {
		return errors.New("categoría del relevo incompatible con sus integrantes")
	}
	return nil
}
func saveMark(ctx context.Context, tx *sql.Tx, m domain.Mark, importing bool) (bool, error) {
	if e := validateRelations(ctx, tx, m); e != nil {
		return false, e
	}
	if m.ID != "" && !importing {
		var old domain.Mark
		var b string
		if e := tx.QueryRowContext(ctx, "SELECT payload FROM marks WHERE id=?", m.ID).Scan(&b); e != nil {
			return false, errors.New("marca no encontrada")
		}
		if e := json.Unmarshal([]byte(b), &old); e != nil {
			return false, e
		}
		// The immutable import fingerprint keeps later imports from restoring an edited source row.
		m.Source = old.Source
		m.SourceURL = old.SourceURL
		m.RetrievedAt = old.RetrievedAt
		if e := history(ctx, tx, "marks", "id", m.ID); e != nil {
			return false, e
		}
		_, e := tx.ExecContext(ctx, "UPDATE marks SET club_id=?,license=?,payload=? WHERE id=?", m.ClubID, nullable(m.License), payload(m), m.ID)
		return false, e
	}
	m.ID = uuid.NewString()
	r, e := tx.ExecContext(ctx, "INSERT INTO marks(id,fingerprint,club_id,license,payload) VALUES(?,?,?,?,?) ON CONFLICT(fingerprint) DO NOTHING", m.ID, fingerprint(m), m.ClubID, nullable(m.License), payload(m))
	if e != nil {
		return false, e
	}
	n, e := r.RowsAffected()
	return n > 0, e
}
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func (s *Store) SaveMark(ctx context.Context, m domain.Mark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if m.ID == "" {
		m.Source = "manual"
		m.SourceURL = ""
	}
	if _, e = saveMark(ctx, tx, m, false); e != nil {
		return e
	}
	return tx.Commit()
}

type ImportResult struct {
	Inserted   int `json:"inserted"`
	Duplicates int `json:"duplicates"`
}

func (s *Store) Import(ctx context.Context, d domain.Data, metadata ...any) (ImportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := ImportResult{}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	// Existing local identities and club assignments always win.
	for _, c := range d.Clubs {
		if suppressed(ctx, tx, "clubs", c.ID) {
			continue
		}
		if e = c.Validate(); e != nil {
			return out, e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO clubs(id,payload) VALUES(?,?) ON CONFLICT(id) DO NOTHING", c.ID, payload(c)); e != nil {
			return out, e
		}
	}
	excludedPeople := map[string]bool{}
	for _, v := range d.Swimmers {
		if suppressed(ctx, tx, "swimmers", v.License) || suppressed(ctx, tx, "clubs", v.ClubID) {
			excludedPeople[v.License] = true
			continue
		}
		if e = v.Validate(); e != nil {
			return out, e
		}
		var old string
		e = tx.QueryRowContext(ctx, "SELECT payload FROM swimmers WHERE license=?", v.License).Scan(&old)
		if e == nil {
			var existing domain.Swimmer
			if e = json.Unmarshal([]byte(old), &existing); e != nil {
				return out, e
			}
			if existing.BirthYear != v.BirthYear || existing.Gender != v.Gender {
				return out, fmt.Errorf("conflicto de identidad en licencia %s; revisar antes de importar", v.License)
			}
			continue
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return out, e
		}
		if e = saveSwimmer(ctx, tx, v); e != nil {
			return out, e
		}
	}
	for _, m := range d.Marks {
		if excludedMark(ctx, tx, m) || excludedPeople[m.License] {
			out.Duplicates++
			continue
		}
		if m.Source != "fegan" && m.Source != "fegan-pdf" && m.Source != "local-pdf" {
			return out, errors.New("procedencia de importación inválida")
		}
		inserted, e := saveMark(ctx, tx, m, true)
		if e != nil {
			return out, e
		}
		if inserted {
			out.Inserted++
		} else {
			out.Duplicates++
		}
	}
	if len(metadata) > 0 {
		if _, e = tx.ExecContext(ctx, "INSERT INTO history(entity,entity_id,before) VALUES('import',?,?)", uuid.NewString(), payload(map[string]any{"metadata": metadata[0], "result": out})); e != nil {
			return out, e
		}
	}
	return out, tx.Commit()
}
func (s *Store) Backup(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, e := os.CreateTemp("", "fegan-backup-*.sqlite")
	if e != nil {
		return nil, e
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	defer os.Remove(path)
	_, e = s.db.ExecContext(ctx, "VACUUM INTO '"+strings.ReplaceAll(path, "'", "''")+"'")
	if e != nil {
		return nil, e
	}
	return os.ReadFile(path)
}
func (s *Store) History(ctx context.Context) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, e := s.db.QueryContext(ctx, "SELECT entity,entity_id,before,at FROM history ORDER BY id DESC LIMIT 500")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var entity, id, before, at string
		if e = rows.Scan(&entity, &id, &before, &at); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"entity": entity, "id": id, "before": json.RawMessage(before), "at": at})
	}
	return out, rows.Err()
}

func (s *Store) Setting(ctx context.Context, key string, out any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var value string
	e := s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", key).Scan(&value)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal([]byte(value), out)
}
func (s *Store) SetSetting(ctx context.Context, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, e := s.db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, payload(value))
	return e
}
