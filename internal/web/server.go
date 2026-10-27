package web

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"github.com/marianoao/fegan-master-team/internal/fegan"
	"github.com/marianoao/fegan-master-team/internal/store"
	"github.com/marianoao/fegan-master-team/internal/syncer"
)

//go:embed static/*
var assets embed.FS

type preview struct {
	result  fegan.Preview
	expires time.Time
}
type Server struct {
	browser    *browserLifecycle
	Sync       *syncer.Service
	Store      *store.Store
	Fegan      *fegan.Client
	Host       string
	token      string
	mu         sync.Mutex
	operations sync.RWMutex
	resetMu    sync.Mutex
	workCtx    context.Context
	workCancel context.CancelFunc
	previews   map[string]preview
	imports    chan struct{}
	jobs       map[string]importJob
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func New(s *store.Store, host string) *Server {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	client := fegan.New()
	ctx, cancel := context.WithCancel(context.Background())
	workCtx, workCancel := context.WithCancel(ctx)
	return &Server{Store: s, Fegan: client, Sync: syncer.New(s, client), Host: host, token: hex.EncodeToString(b), previews: map[string]preview{}, imports: make(chan struct{}, 1), jobs: map[string]importJob{}, ctx: ctx, cancel: cancel, workCtx: workCtx, workCancel: workCancel}
}
func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, e error, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	respond(w, map[string]string{"error": e.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("datos JSON no válidos")
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		return errors.New("se esperaba un único objeto JSON")
	}
	return nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.setupRoutes(mux)
	s.editRoutes(mux)
	mux.HandleFunc("POST /api/reset", s.resetDatabase)
	mux.HandleFunc("POST /api/browser/presence", s.browserPresence)
	mux.HandleFunc("GET /api/session", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		token := s.token
		s.mu.Unlock()
		respond(w, map[string]any{"token": token, "events": domain.Events, "auto_close": s.browser != nil})
	})
	mux.HandleFunc("GET /api/data", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.Store.Snapshot(r.Context())
		if e != nil {
			failure(w, e, 500)
			return
		}
		respond(w, v)
	})
	mux.HandleFunc("POST /api/clubs", func(w http.ResponseWriter, r *http.Request) {
		var v domain.Club
		if e := decode(w, r, &v); e != nil {
			failure(w, e, 400)
			return
		}
		if e := s.Store.SaveClub(r.Context(), v); e != nil {
			failure(w, e, 400)
			return
		}
		respond(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /api/swimmers", func(w http.ResponseWriter, r *http.Request) {
		var v domain.Swimmer
		if e := decode(w, r, &v); e != nil {
			failure(w, e, 400)
			return
		}
		if e := s.Store.SaveSwimmer(r.Context(), v); e != nil {
			failure(w, e, 400)
			return
		}
		s.scheduleRivals()
		respond(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /api/marks", func(w http.ResponseWriter, r *http.Request) {
		var v domain.Mark
		if e := decode(w, r, &v); e != nil {
			failure(w, e, 400)
			return
		}
		if e := s.Store.SaveMark(r.Context(), v); e != nil {
			failure(w, e, 400)
			return
		}
		s.scheduleRivals()
		respond(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.Store.History(r.Context())
		if e != nil {
			failure(w, e, 500)
			return
		}
		respond(w, v)
	})
	mux.HandleFunc("GET /api/backup", func(w http.ResponseWriter, r *http.Request) {
		b, e := s.Store.Backup(r.Context())
		if e != nil {
			failure(w, e, 500)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.sqlite3")
		w.Header().Set("Content-Disposition", `attachment; filename="fegan-backup.sqlite"`)
		w.Write(b)
	})
	mux.HandleFunc("POST /api/import/preview", func(w http.ResponseWriter, r *http.Request) {
		select {
		case s.imports <- struct{}{}:
			defer func() { <-s.imports }()
		default:
			failure(w, errors.New("ya hay una consulta FEGAN en curso"), 409)
			return
		}
		var q fegan.Query
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
		defer cancel()
		v, e := s.Fegan.Preview(ctx, q)
		if e != nil {
			failure(w, e, 422)
			return
		}
		id := uuid.NewString()
		s.mu.Lock()
		for k, p := range s.previews {
			if time.Now().After(p.expires) {
				delete(s.previews, k)
			}
		}
		if len(s.previews) >= 8 {
			s.previews = map[string]preview{}
		}
		s.previews[id] = preview{v, time.Now().Add(15 * time.Minute)}
		s.mu.Unlock()
		respond(w, map[string]any{"id": id, "preview": v})
	})
	mux.HandleFunc("POST /api/import/confirm", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			ID string `json:"id"`
		}
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		s.mu.Lock()
		p, ok := s.previews[q.ID]
		s.mu.Unlock()
		if !ok || time.Now().After(p.expires) {
			failure(w, errors.New("vista previa caducada; repetir consulta"), 400)
			return
		}
		v, e := s.Store.Import(r.Context(), p.result.Data, map[string]any{"query": p.result.Query, "retrieved_at": p.result.RetrievedAt, "requests": p.result.Requests})
		if e != nil {
			failure(w, e, 400)
			return
		}
		s.scheduleRivals()
		s.mu.Lock()
		delete(s.previews, q.ID)
		s.mu.Unlock()
		respond(w, v)
	})
	mux.HandleFunc("POST /api/recommend", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			domain.Filter
			License string `json:"license"`
		}
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		d, e := s.Store.Snapshot(r.Context())
		if e != nil {
			failure(w, e, 500)
			return
		}
		if q.SeasonStart == 0 {
			q.SeasonStart = domain.SeasonStartOn(time.Now())
		}
		// Opportunity references always cover the last two years; the own best
		// remains historical and selects a single timing per event.
		now := time.Now()
		q.From = now.AddDate(-2, 0, 0).Format("2006-01-02")
		q.To = now.Format("2006-01-02")
		q.Chrono = ""
		v, e := domain.Recommend(d, q.License, q.Filter)
		if e != nil {
			failure(w, e, 400)
			return
		}
		respond(w, v)
	})
	mux.HandleFunc("POST /api/relays", func(w http.ResponseWriter, r *http.Request) {
		var q domain.RelayRequest
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		d, e := s.Store.Snapshot(r.Context())
		if e != nil {
			failure(w, e, 500)
			return
		}
		if q.Pool == 0 || q.Chrono == "" {
			groups := []relayComparison{}
			pools := []int{q.Pool}
			if q.Pool == 0 {
				pools = []int{25, 50}
			}
			chronos := []string{q.Chrono}
			if q.Chrono == "" {
				chronos = []string{""}
			}
			for _, pool := range pools {
				for _, chrono := range chronos {
					v := q
					v.Pool = pool
					v.Chrono = chrono
					advice, err := domain.RecommendRelayCategories(r.Context(), d, v)
					if err != nil {
						failure(w, err, 400)
						return
					}
					groups = append(groups, relayComparison{pool, chrono, advice})
				}
			}
			respond(w, groups)
			return
		}
		v, e := domain.RecommendRelayCategories(r.Context(), d, q)
		if e != nil {
			failure(w, e, 400)
			return
		}
		respond(w, v)
	})
	static, _ := fs.Sub(assets, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.Host != s.Host {
			failure(w, errors.New("host local no válido"), 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+s.Host {
			failure(w, errors.New("origen no permitido"), 403)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			failure(w, errors.New("petición externa no permitida"), 403)
			return
		}
		// Presence must stay connected across a reset; other API requests belong
		// to the current data generation and are canceled before it is cleared.
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/reset" && r.URL.Path != "/api/browser/presence" {
			s.mu.Lock()
			workCtx := s.workCtx
			s.mu.Unlock()
			ctx, cancel := context.WithCancel(r.Context())
			stop := context.AfterFunc(workCtx, cancel)
			defer cancel()
			defer stop()
			r = r.WithContext(ctx)
			s.operations.RLock()
			defer s.operations.RUnlock()
			if workCtx.Err() != nil || ctx.Err() != nil {
				failure(w, errors.New("los datos se están reiniciando; recarga la página"), 409)
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" && !s.validToken(r) {
			failure(w, errors.New("sesión local no válida; recarga la página"), 403)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) Close() {
	s.cancel()
	if s.browser != nil {
		s.browser.close()
	}
	s.resetMu.Lock()
	defer s.resetMu.Unlock()
	s.Sync.Suspend()
	s.operations.Lock()
	defer s.operations.Unlock()
	s.wg.Wait()
	s.Sync.Close()
}
