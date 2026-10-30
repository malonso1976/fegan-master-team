package syncer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/fegan"
	"github.com/malonso1976/fegan-master-team/internal/store"
)

const clubHistoryYears = 5

type fullSyncStamp struct {
	ClubID       string
	Date         string
	HistoryYears int
}

type Status struct {
	Reused     int      `json:"reused"`
	Scope      string   `json:"scope"`
	Running    bool     `json:"running"`
	Stage      string   `json:"stage"`
	Completed  int      `json:"completed"`
	Total      int      `json:"total"`
	Inserted   int      `json:"inserted"`
	Duplicates int      `json:"duplicates"`
	Warnings   []string `json:"warnings"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at"`
	ClubID     string   `json:"club_id"`
}
type Service struct {
	store     *store.Store
	client    *fegan.Client
	mu        sync.Mutex
	status    Status
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	pending   bool
	suspended bool
}

func New(s *store.Store, c *fegan.Client) *Service {
	service := &Service{store: s, client: c}
	_ = s.Setting(context.Background(), "sync_status", &service.status)
	service.status.Running = false
	return service
}
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.status
	v.Warnings = append([]string{}, v.Warnings...)
	return v
}
func (s *Service) Close() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// Suspend prevents new work and waits for existing writers to stop. A reset
// replaces the service afterwards; ordinary Close remains reusable for club changes.
func (s *Service) Suspend() {
	s.mu.Lock()
	s.suspended = true
	s.pending = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}
func (s *Service) Start(club, scope string, force bool) Status {
	s.mu.Lock()
	if s.suspended {
		v := s.status
		s.mu.Unlock()
		return v
	}
	if s.status.Running {
		if scope == "rivals" {
			s.pending = true
		}
		v := s.status
		s.mu.Unlock()
		return v
	}
	var full fullSyncStamp
	if scope == "all" && !force {
		_ = s.store.Setting(context.Background(), "full_sync_done", &full)
	}
	if scope == "all" && !force && full.ClubID == club && full.Date == time.Now().Format("2006-01-02") && full.HistoryYears == clubHistoryYears {
		v := s.status
		s.mu.Unlock()
		return v
	}
	s.status = Status{Scope: scope, Running: true, ClubID: club, Stage: "Preparando fuentes", StartedAt: time.Now().UTC().Format(time.RFC3339), Warnings: []string{}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	s.cancel = cancel
	s.wg.Add(1)
	v := s.status
	s.mu.Unlock()
	go func() {
		defer s.wg.Done()
		defer cancel()
		if force {
			s.client.RefreshCaches(scope == "all" || scope == "rivals", scope == "all" || scope == "relays")
		}
		s.run(ctx, club, scope, force)
		s.mu.Lock()
		s.status.Running = false
		s.status.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.status.Stage = "Actualización finalizada"
		status := s.status
		pending := s.pending
		s.pending = false
		s.mu.Unlock()
		_ = s.store.SetSetting(context.Background(), "sync_status", status)
		if scope == "all" && len(status.Warnings) == 0 {
			_ = s.store.SetSetting(context.Background(), "full_sync_done", fullSyncStamp{ClubID: club, Date: time.Now().Format("2006-01-02"), HistoryYears: clubHistoryYears})
		}
		if pending && ctx.Err() == nil {
			s.Start(club, "rivals", false)
		}
	}()
	return v
}
func (s *Service) stage(name string, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Stage = name
	s.status.Completed = 0
	s.status.Total = total
}
func (s *Service) advance() { s.mu.Lock(); s.status.Completed++; s.mu.Unlock() }
func (s *Service) warn(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.status.Warnings) < 100 {
		s.status.Warnings = append(s.status.Warnings, message)
	}
}
func (s *Service) save(ctx context.Context, d domain.Data, metadata any) bool {
	result, e := s.store.Import(ctx, d, metadata)
	if e != nil {
		s.warn(e.Error())
		return false
	}
	s.mu.Lock()
	s.status.Inserted += result.Inserted
	s.status.Duplicates += result.Duplicates
	s.mu.Unlock()
	return true
}
func (s *Service) run(ctx context.Context, club, scope string, force bool) {
	now := time.Now()
	season := domain.SeasonStartOn(now)
	if scope == "all" {
		s.stage("Catálogo de clubes de la temporada", 1)
		catalog, e := s.client.ClubCatalog(ctx, season)
		if e != nil {
			s.warn("Catálogo: " + e.Error())
		} else if e = s.store.SetSetting(ctx, "club_catalog", catalog); e != nil {
			s.warn(e.Error())
		}
		s.advance()
		if !s.loadClubHistory(ctx, club, now) {
			return
		}
		if !s.loadPersonalBests(ctx, club, now) {
			return
		}
	}
	if scope != "relays" {
		state, e := s.store.Snapshot(ctx)
		if e != nil {
			s.warn(e.Error())
			return
		}
		queries := fegan.RivalQueries(state, club, season, now)
		s.stage("Rivales · diez mejores por grupo y prueba", len(queries))
		if len(queries) == 0 {
			s.warn("Sin marcas del equipo para determinar pruebas relevantes; añade marcas o sincroniza el club")
		}
		type queryCache struct {
			Date    string
			Entries map[string]bool
		}
		cache := queryCache{}
		_ = s.store.Setting(ctx, "rival_cache", &cache)
		today := now.Format("2006-01-02")
		if cache.Date != today || cache.Entries == nil {
			cache = queryCache{Date: today, Entries: map[string]bool{}}
		}
		for _, q := range queries {
			key := fmt.Sprintf("%+v", q)
			if !force && cache.Entries[key] {
				s.mu.Lock()
				s.status.Reused++
				s.mu.Unlock()
				s.advance()
				continue
			}
			if ctx.Err() != nil {
				s.warn("Actualización interrumpida o tiempo agotado")
				return
			}
			p, e := s.client.Preview(ctx, q)
			if e != nil {
				s.warn(fmt.Sprintf("Rivales %s/%dm/%s: %s", q.Event, q.Pool, q.Chrono, e))
			} else {
				if s.save(ctx, p.Data, map[string]any{"automatic": "top10", "query": q, "requests": p.Requests, "warnings": p.Warnings, "birth_from": q.BirthFrom, "birth_to": q.BirthTo}) {
					cache.Entries[key] = true
					if e = s.store.SetSetting(ctx, "rival_cache", cache); e != nil {
						s.warn(e.Error())
					}
				}
			}
			s.advance()
		}
	}
	if scope == "all" || scope == "relays" {
		var catalog fegan.Catalog
		if e := s.store.Setting(ctx, "club_catalog", &catalog); e != nil {
			s.warn(e.Error())
			return
		}
		if len(catalog.Clubs) == 0 {
			state, e := s.store.Snapshot(ctx)
			if e != nil {
				s.warn(e.Error())
				return
			}
			catalog.Clubs = state.Clubs
		}
		s.stage("Localizando campeonatos · dos temporadas", 0)
		championships, e := s.client.Championships(ctx, now)
		if e != nil {
			s.warn(e.Error())
			return
		}
		s.stage("PDF de relevos · invierno y verano", len(championships))
		for _, championship := range championships {
			if ctx.Err() != nil {
				s.warn("Actualización interrumpida o tiempo agotado")
				return
			}
			result, e := s.client.RelayChampionship(ctx, championship, catalog.Clubs)
			if e != nil {
				s.warn(e.Error())
			} else {
				for _, warning := range result.Warnings {
					s.warn(warning)
				}
				s.save(ctx, result.Data, map[string]any{"automatic": "relay-pdf", "championship": championship})
			}
			s.advance()
		}
	}
}

func clubHistoryQueries(club string, now time.Time) []fegan.Query {
	queries := []fegan.Query{}
	for _, gender := range []string{"M", "F"} {
		for _, pool := range []int{25, 50} {
			for _, chrono := range []string{"E", "M"} {
				for _, event := range domain.Events {
					if domain.Relay(event) {
						continue
					}
					queries = append(queries, fegan.Query{ClubID: club, Member: true, Gender: gender, Pool: pool, Chrono: chrono, Event: event, From: now.AddDate(-clubHistoryYears, 0, 0).Format("2006-01-02"), To: now.Format("2006-01-02")})
				}
			}
		}
	}
	return queries
}

func (s *Service) loadClubHistory(ctx context.Context, club string, now time.Time) bool {
	queries := clubHistoryQueries(club, now)
	s.stage("Nadadores y marcas del club · últimos cinco años", len(queries))
	for _, q := range queries {
		if ctx.Err() != nil {
			s.warn("Actualización interrumpida o tiempo agotado")
			return false
		}
		p, e := s.client.Preview(ctx, q)
		if e != nil {
			s.warn(fmt.Sprintf("Club %s/%dm/%s: %s", q.Event, q.Pool, q.Chrono, e))
		} else {
			for _, warning := range p.Warnings {
				if strings.HasPrefix(warning, "Se omitieron") {
					s.warn(fmt.Sprintf("Club %s/%dm/%s: %s", q.Event, q.Pool, q.Chrono, warning))
				}
			}
			if len(p.Data.Marks) > 0 {
				s.save(ctx, p.Data, map[string]any{"automatic": "team", "query": q, "history_years": clubHistoryYears})
			}
		}
		s.advance()
	}
	return true
}

func (s *Service) loadPersonalBests(ctx context.Context, club string, now time.Time) bool {
	state, err := s.store.Snapshot(ctx)
	if err != nil {
		s.warn("No se pudo leer el equipo para consultar mejores marcas personales")
		return false
	}
	members := []domain.Swimmer{}
	for _, swimmer := range state.Swimmers {
		if swimmer.ClubID == club && swimmer.Member {
			members = append(members, swimmer)
		}
	}
	s.stage("Mejores marcas personales · últimos cinco años", len(members))
	for _, swimmer := range members {
		if ctx.Err() != nil {
			s.warn("Actualización de mejores marcas personales interrumpida o tiempo agotado")
			return false
		}
		q := fegan.Query{License: swimmer.License, Gender: swimmer.Gender, BirthFrom: swimmer.BirthYear, BirthTo: swimmer.BirthYear, Event: "all", From: now.AddDate(-clubHistoryYears, 0, 0).Format("2006-01-02"), To: now.Format("2006-01-02")}
		p, err := s.client.PersonalPreview(ctx, q)
		if err != nil {
			s.warn("No se pudo completar una consulta de mejores marcas personales; cobertura parcial.")
		} else if len(p.Data.Marks) > 0 {
			s.save(ctx, p.Data, map[string]any{"automatic": "personal-bests", "query": q, "requests": p.Requests})
		}
		s.advance()
	}
	return true
}
