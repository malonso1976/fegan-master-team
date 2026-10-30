package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/malonso1976/fegan-master-team/internal/domain"
)

func suppressionKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}
func suppressed(ctx context.Context, tx *sql.Tx, kind, id string) bool {
	var n int
	_ = tx.QueryRowContext(ctx, "SELECT count(*) FROM suppressed WHERE kind=? AND key=?", kind, suppressionKey(id)).Scan(&n)
	return n > 0
}
func excludedMark(ctx context.Context, tx *sql.Tx, m domain.Mark) bool {
	if suppressed(ctx, tx, "clubs", m.ClubID) || suppressed(ctx, tx, "marks", fingerprint(m)) || (m.License != "" && suppressed(ctx, tx, "swimmers", m.License)) {
		return true
	}
	for _, l := range m.RelayLicenses {
		if suppressed(ctx, tx, "swimmers", l) {
			return true
		}
	}
	return false
}

type DeleteResult struct {
	Clubs       int  `json:"clubs"`
	Swimmers    int  `json:"swimmers"`
	Marks       int  `json:"marks"`
	HomeCleared bool `json:"home_cleared"`
}

// Delete removes dependent marks, including JSON relay rosters, in one transaction.
// Hashed exclusions prevent a concurrent or subsequent import undoing the deletion.
func (s *Store) Delete(ctx context.Context, kind, id string) (DeleteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := DeleteResult{}
	if kind != "clubs" && kind != "swimmers" && kind != "marks" {
		return out, errors.New("tipo de borrado no válido")
	}
	d, e := s.snapshot(ctx)
	if e != nil {
		return out, e
	}
	people := map[string]bool{}
	found := false
	for _, c := range d.Clubs {
		if kind == "clubs" && c.ID == id {
			found = true
			out.Clubs++
		}
	}
	for _, p := range d.Swimmers {
		if (kind == "swimmers" && p.License == id) || (kind == "clubs" && p.ClubID == id) {
			people[p.License] = true
			out.Swimmers++
			if kind == "swimmers" {
				found = true
			}
		}
	}
	for _, m := range d.Marks {
		if kind == "marks" && m.ID == id {
			found = true
		}
	}
	if !found {
		return out, errors.New("registro no encontrado")
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	suppress := func(k, v string) error {
		_, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO suppressed(kind,key) VALUES(?,?)", k, suppressionKey(v))
		return err
	}
	remove := func(table, key, value string) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM history WHERE entity=? AND entity_id=?", table, value); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE "+key+"=?", value)
		return err
	}
	for _, m := range d.Marks {
		affected := (kind == "marks" && m.ID == id) || (kind == "clubs" && m.ClubID == id) || people[m.License]
		for _, l := range m.RelayLicenses {
			affected = affected || people[l]
		}
		if !affected {
			continue
		}
		var original string
		if e = tx.QueryRowContext(ctx, "SELECT fingerprint FROM marks WHERE id=?", m.ID).Scan(&original); e != nil {
			return out, e
		}
		if e = suppress("marks", original); e != nil {
			return out, e
		}
		if e = suppress("marks", fingerprint(m)); e != nil {
			return out, e
		}
		if e = remove("marks", "id", m.ID); e != nil {
			return out, e
		}
		out.Marks++
	}
	for l := range people {
		if e = suppress("swimmers", l); e != nil {
			return out, e
		}
		if e = remove("swimmers", "license", l); e != nil {
			return out, e
		}
	}
	if kind == "clubs" {
		if e = suppress("clubs", id); e != nil {
			return out, e
		}
		if e = remove("clubs", "id", id); e != nil {
			return out, e
		}
		var home string
		if e = tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='home_club'").Scan(&home); e != nil && e != sql.ErrNoRows {
			return out, e
		}
		if home == payload(id) {
			out.HomeCleared = true
			if _, e = tx.ExecContext(ctx, "DELETE FROM settings WHERE key IN ('home_club','full_sync_done')"); e != nil {
				return out, e
			}
		}
	}
	// Import summaries can contain the deleted license in their original query.
	if _, e = tx.ExecContext(ctx, "DELETE FROM history WHERE entity='import'"); e != nil {
		return out, e
	}
	return out, tx.Commit()
}
