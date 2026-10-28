package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/marianoao/fegan-master-team/internal/domain"
)

// DeactivateSwimmers preserves profiles and marks; the whole selection commits together.
func (s *Store) DeactivateSwimmers(ctx context.Context, licenses []string) (int, error) {
	return s.setSwimmerTeamState(ctx, licenses, false)
}

// ActivateSwimmers makes existing selected profiles team members available for relays.
func (s *Store) ActivateSwimmers(ctx context.Context, licenses []string) (int, error) {
	return s.setSwimmerTeamState(ctx, licenses, true)
}

func (s *Store) setSwimmerTeamState(ctx context.Context, licenses []string, active bool) (int, error) {
	if len(licenses) == 0 || len(licenses) > 1000 {
		return 0, errors.New("selecciona entre 1 y 1000 nadadores")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	seen := map[string]bool{}
	changed := 0
	for _, license := range licenses {
		if !domain.ValidID(license) {
			return 0, errors.New("selección de nadadores no válida")
		}
		if seen[license] {
			continue
		}
		seen[license] = true
		var raw string
		if err = tx.QueryRowContext(ctx, "SELECT payload FROM swimmers WHERE license=?", license).Scan(&raw); err != nil {
			return 0, errors.New("algún nadador seleccionado ya no existe; actualiza el listado")
		}
		var swimmer domain.Swimmer
		if err = json.Unmarshal([]byte(raw), &swimmer); err != nil {
			return 0, err
		}
		if swimmer.Member == active && swimmer.Available == active {
			continue
		}
		if err = history(ctx, tx, "swimmers", "license", license); err != nil {
			return 0, err
		}
		swimmer.Member = active
		swimmer.Available = active
		if err = saveSwimmer(ctx, tx, swimmer); err != nil {
			return 0, err
		}
		changed++
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return changed, nil
}
