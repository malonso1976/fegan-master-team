package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"github.com/marianoao/fegan-master-team/internal/fegan"
)

type relayComparison struct {
	Pool   int                `json:"pool"`
	Chrono string             `json:"chrono"`
	Advice domain.RelayAdvice `json:"advice"`
}
type importJob struct {
	Ready   bool           `json:"ready"`
	Error   string         `json:"error,omitempty"`
	ID      string         `json:"id"`
	Preview *fegan.Preview `json:"preview,omitempty"`
}

func (s *Server) retain(p fegan.Preview) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.previews {
		if time.Now().After(v.expires) {
			delete(s.previews, k)
		}
	}
	if len(s.previews) >= 8 {
		s.previews = map[string]preview{}
	}
	id := uuid.NewString()
	s.previews[id] = preview{p, time.Now().Add(15 * time.Minute)}
	return id
}
func (s *Server) editRoutes(mux *http.ServeMux) {
	for _, active := range []bool{false, true} {
		path := "POST /api/swimmers/deactivate"
		if active {
			path = "POST /api/swimmers/activate"
		}
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			var q struct {
				Licenses []string `json:"licenses"`
			}
			if err := decode(w, r, &q); err != nil {
				failure(w, err, 400)
				return
			}
			var changed int
			var err error
			if active {
				changed, err = s.Store.ActivateSwimmers(r.Context(), q.Licenses)
			} else {
				changed, err = s.Store.DeactivateSwimmers(r.Context(), q.Licenses)
			}
			if err != nil {
				failure(w, err, 400)
				return
			}
			respond(w, map[string]int{"updated": changed})
		})
	}

	mux.HandleFunc("POST /api/delete", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		}
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		result, e := s.Store.Delete(r.Context(), q.Kind, q.ID)
		if e != nil {
			failure(w, e, 400)
			return
		}
		if result.HomeCleared {
			s.Sync.Close()
		}
		respond(w, result)
	})
	mux.HandleFunc("POST /api/import/start", func(w http.ResponseWriter, r *http.Request) {
		var q fegan.Query
		if e := decode(w, r, &q); e != nil {
			failure(w, e, 400)
			return
		}
		if q.License != "" && q.Gender == "" {
			state, err := s.Store.Snapshot(r.Context())
			if err != nil {
				failure(w, err, 500)
				return
			}
			for _, swimmer := range state.Swimmers {
				if swimmer.License == q.License {
					q.Gender = swimmer.Gender
					break
				}
			}
		}
		if _, e := fegan.ExpandQuery(q); e != nil {
			failure(w, e, 400)
			return
		}
		select {
		case s.imports <- struct{}{}:
		default:
			failure(w, errors.New("ya hay una consulta FEGAN en curso"), 409)
			return
		}
		id := uuid.NewString()
		s.mu.Lock()
		s.jobs = map[string]importJob{id: {ID: id}}
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.imports }()
			ctx, cancel := context.WithTimeout(s.workCtx, 30*time.Minute)
			defer cancel()
			p, e := s.Fegan.WidePreview(ctx, q)
			job := importJob{Ready: true, ID: id}
			if e != nil {
				job.Error = e.Error()
			} else {
				job.ID = s.retain(p)
				job.Preview = &p
			}
			s.mu.Lock()
			s.jobs[id] = job
			s.mu.Unlock()
		}()
		respond(w, importJob{ID: id})
	})
	mux.HandleFunc("GET /api/import/status/{id}", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		job, ok := s.jobs[r.PathValue("id")]
		s.mu.Unlock()
		if !ok {
			failure(w, errors.New("consulta no disponible"), 404)
			return
		}
		respond(w, job)
	})
	mux.HandleFunc("POST /api/relay-pdf/preview", s.localPDF)
}
func (s *Server) localPDF(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	defer r.Body.Close()
	if e := r.ParseMultipartForm(21 << 20); e != nil {
		failure(w, errors.New("sube un PDF de hasta 20 MiB"), 400)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, e := r.FormFile("file")
	if e != nil {
		failure(w, errors.New("selecciona un PDF de resultados"), 400)
		return
	}
	defer file.Close()
	b, e := io.ReadAll(io.LimitReader(file, (20<<20)+1))
	if e != nil || len(b) > 20<<20 {
		failure(w, errors.New("PDF superior a 20 MiB"), 400)
		return
	}
	date := r.FormValue("date")
	if _, e = time.Parse("2006-01-02", date); e != nil {
		failure(w, errors.New("indica la fecha del campeonato"), 400)
		return
	}
	kind := r.FormValue("championship")
	location := "Campeonato Gallego Máster de Invierno"
	if kind == "summer" {
		location = "Campeonato Gallego Máster de Verano"
	} else if kind != "winter" {
		failure(w, errors.New("selecciona campeonato de invierno o verano"), 400)
		return
	}
	chrono := r.FormValue("chrono")
	if chrono != "E" && chrono != "M" {
		failure(w, errors.New("indica el cronometraje del documento"), 400)
		return
	}
	d, e := s.Store.Snapshot(r.Context())
	if e != nil {
		failure(w, e, 500)
		return
	}
	var catalog fegan.Catalog
	if e = s.Store.Setting(r.Context(), "club_catalog", &catalog); e != nil {
		failure(w, e, 500)
		return
	}
	known := map[string]domain.Club{}
	for _, c := range catalog.Clubs {
		known[c.ID] = c
	}
	for _, c := range d.Clubs {
		known[c.ID] = c
	}
	aliases := map[string]string{}
	if raw := r.FormValue("aliases"); raw != "" {
		if e = json.Unmarshal([]byte(raw), &aliases); e != nil {
			failure(w, errors.New("alias no válidos"), 400)
			return
		}
	}
	for alias, id := range aliases {
		if _, ok := known[id]; !ok || strings.TrimSpace(alias) == "" {
			failure(w, errors.New("cada alias debe indicar un código de club conocido"), 400)
			return
		}
	}
	clubs := []domain.Club{}
	for _, c := range known {
		clubs = append(clubs, c)
	}
	sum := sha256.Sum256(b)
	source := fmt.Sprintf("local-pdf:%s/%s", hex.EncodeToString(sum[:]), filepath.Base(header.Filename))
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	result, e := fegan.ParseRelayPDF(ctx, b, source, date, location, clubs, aliases)
	if e != nil {
		failure(w, e, 422)
		return
	}
	for i := range result.Data.Marks {
		m := &result.Data.Marks[i]
		m.Source = "local-pdf"
		m.Verified = r.FormValue("verified") == "true"
		m.Chrono = chrono
	}
	p := fegan.Preview{Data: result.Data, Warnings: result.Warnings, RetrievedAt: time.Now().UTC().Format(time.RFC3339)}
	p.Warnings = append(p.Warnings, "PDF aportado por el usuario. Revisa clubes, fechas, categorías y cronometraje antes de guardar. No se consultó FEGAN.")
	respond(w, map[string]any{"id": s.retain(p), "preview": p})
}
