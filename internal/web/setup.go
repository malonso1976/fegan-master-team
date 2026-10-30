package web

import (
	"context"
	"errors"
	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/fegan"
	"net/http"
	"time"
)

func (s *Server) scheduleRivals() {
	var club string
	if s.Store.Setting(context.Background(), "home_club", &club) == nil && club != "" {
		s.Sync.Start(club, "rivals", false)
	}
}
func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/setup", func(w http.ResponseWriter, r *http.Request) {
		var club string
		var catalog fegan.Catalog
		if e := s.Store.Setting(r.Context(), "home_club", &club); e != nil {
			failure(w, e, 500)
			return
		}
		if e := s.Store.Setting(r.Context(), "club_catalog", &catalog); e != nil {
			failure(w, e, 500)
			return
		}
		respond(w, map[string]any{"home_club_id": club, "catalog": catalog, "sync": s.Sync.Status()})
	})
	mux.HandleFunc("POST /api/catalog", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		catalog, e := s.Fegan.ClubCatalog(ctx, domain.SeasonStartOn(time.Now()))
		if e != nil {
			failure(w, e, 422)
			return
		}
		if e = s.Store.SetSetting(ctx, "club_catalog", catalog); e != nil {
			failure(w, e, 500)
			return
		}
		respond(w, catalog)
	})
	mux.HandleFunc("POST /api/setup", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			ClubID string `json:"club_id"`
		}
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		if status := s.Sync.Status(); status.Running && status.ClubID != q.ClubID {
			failure(w, errors.New("espera a que termine la actualización del club actual antes de cambiarlo"), 409)
			return
		}
		var catalog fegan.Catalog
		if e := s.Store.Setting(r.Context(), "club_catalog", &catalog); e != nil {
			failure(w, e, 500)
			return
		}
		var chosen *domain.Club
		for _, c := range catalog.Clubs {
			if c.ID == q.ClubID {
				v := c
				chosen = &v
				break
			}
		}
		{ // Prefer the saved club name and local corrections over catalog data.
			state, e := s.Store.Snapshot(r.Context())
			if e != nil {
				failure(w, e, 500)
				return
			}
			for _, c := range state.Clubs {
				if c.ID == q.ClubID {
					v := c
					chosen = &v
					break
				}
			}
		}
		if chosen == nil {
			failure(w, errors.New("selecciona un club válido o créalo manualmente"), 400)
			return
		}
		if e := s.Store.SaveClub(r.Context(), *chosen); e != nil {
			failure(w, e, 400)
			return
		}
		if e := s.Store.SetSetting(r.Context(), "home_club", chosen.ID); e != nil {
			failure(w, e, 500)
			return
		}
		respond(w, s.Sync.Start(chosen.ID, "all", true))
	})
	mux.HandleFunc("GET /api/sync", func(w http.ResponseWriter, r *http.Request) { respond(w, s.Sync.Status()) })
	mux.HandleFunc("POST /api/sync", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Scope string `json:"scope"`
			Force bool   `json:"force"`
		}
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		if q.Scope != "all" && q.Scope != "rivals" && q.Scope != "relays" {
			failure(w, errors.New("tipo de actualización no válido"), 400)
			return
		}
		var club string
		if e := s.Store.Setting(r.Context(), "home_club", &club); e != nil || club == "" {
			failure(w, errors.New("elige primero el club del equipo"), 400)
			return
		}
		respond(w, s.Sync.Start(club, q.Scope, q.Force))
	})
}
