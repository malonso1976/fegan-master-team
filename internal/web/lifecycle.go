package web

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

const browserCloseGrace = 5 * time.Second

type browserLifecycle struct {
	mu         sync.Mutex
	clients    int
	grace      time.Duration
	timer      *time.Timer
	done       chan struct{}
	stopped    bool
	generation uint64
}

func newBrowserLifecycle(grace time.Duration) *browserLifecycle {
	return &browserLifecycle{grace: grace, done: make(chan struct{})}
}

// TrackBrowser must be enabled before serving requests. No timeout starts until
// a page connects: a failed browser launch must leave the printed URL usable.
func (s *Server) TrackBrowser() <-chan struct{} {
	s.browser = newBrowserLifecycle(browserCloseGrace)
	return s.browser.done
}
func (b *browserLifecycle) attach() (func(), bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return nil, false
	}
	if b.timer != nil {
		b.timer.Stop()
	}
	b.clients++
	b.generation++
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.clients--
			if b.clients == 0 && !b.stopped {
				b.generation++
				generation := b.generation
				b.timer = time.AfterFunc(b.grace, func() {
					b.mu.Lock()
					defer b.mu.Unlock()
					if b.clients == 0 && !b.stopped && b.generation == generation {
						b.stopped = true
						close(b.done)
					}
				})
			}
		})
	}, true
}
func (b *browserLifecycle) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Stop()
	}
	if !b.stopped {
		b.stopped = true
		close(b.done)
	}
}
func (s *Server) browserPresence(w http.ResponseWriter, r *http.Request) {
	if s.browser == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	controller := http.NewResponseController(w)
	// This response lasts as long as the page; the ordinary HTTP write timeout
	// still applies to all other routes.
	if e := controller.SetWriteDeadline(time.Time{}); e != nil && !errors.Is(e, http.ErrNotSupported) {
		failure(w, e, 500)
		return
	}
	detach, ok := s.browser.attach()
	if !ok {
		failure(w, errors.New("la aplicación se está cerrando"), 503)
		return
	}
	defer detach()
	w.Header().Set("Content-Type", "text/event-stream")
	if _, e := w.Write([]byte(": connected\n\n")); e != nil {
		return
	}
	if e := controller.Flush(); e != nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.ctx.Done():
			return
		case <-s.browser.done:
			return
		case <-ticker.C:
			if _, e := w.Write([]byte(": alive\n\n")); e != nil {
				return
			}
			if e := controller.Flush(); e != nil {
				return
			}
		}
	}
}
