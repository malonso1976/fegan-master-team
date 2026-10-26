package store

import "context"

// Reset restores the initial logical state atomically. Callers must stop producers
// before invoking it so an old import cannot repopulate the empty database.
func (s *Store) Reset(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, table := range []string{"marks", "swimmers", "clubs", "history", "settings", "suppressed"} {
		if _, e = tx.ExecContext(ctx, "DELETE FROM "+table); e != nil {
			return e
		}
	}
	return tx.Commit()
}
