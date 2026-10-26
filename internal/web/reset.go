package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/marianoao/fegan-master-team/internal/syncer"
)

func (s *Server) validToken(r *http.Request) bool {
	s.mu.Lock()
	token := s.token
	s.mu.Unlock()
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(token)) == 1
}
func (s *Server) resetDatabase(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Confirmation string `json:"confirmation"`
	}
	if e := decode(w, r, &q); e != nil {
		failure(w, e, 400)
		return
	}
	if q.Confirmation != "REINICIAR" {
		failure(w, errors.New("escribe REINICIAR para confirmar el borrado completo"), 400)
		return
	}
	// Allocate the replacement token before changing anything.
	token := make([]byte, 32)
	if _, e := rand.Read(token); e != nil {
		failure(w, e, 500)
		return
	}
	s.resetMu.Lock()
	defer s.resetMu.Unlock()
	if !s.validToken(r) {
		failure(w, errors.New("sesión antigua; recarga la página"), 403)
		return
	}
	s.mu.Lock()
	s.workCancel()
	s.mu.Unlock()
	s.Sync.Suspend() // cancel writers before waiting for HTTP handlers sharing the client
	s.operations.Lock()
	defer s.operations.Unlock()
	s.wg.Wait() // old previews must not be retained after the reset

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
	defer cancel()
	e := s.Store.Reset(ctx)
	s.Fegan.RefreshCaches(true, true)
	s.Sync = syncer.New(s.Store, s.Fegan)
	s.mu.Lock()
	s.previews = map[string]preview{}
	s.jobs = map[string]importJob{}
	s.workCtx, s.workCancel = context.WithCancel(s.ctx)
	if e == nil {
		s.token = hex.EncodeToString(token)
	}
	s.mu.Unlock()
	if e != nil {
		failure(w, e, 500)
		return
	}
	respond(w, map[string]bool{"ok": true})
}
