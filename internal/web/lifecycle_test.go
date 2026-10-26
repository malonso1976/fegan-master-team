package web

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func assertBrowserOpen(t *testing.T, b *browserLifecycle) {
	t.Helper()
	select {
	case <-b.done:
		t.Fatal("browser lifecycle stopped unexpectedly")
	default:
	}
}
func TestBrowserLifetimeReloadAndMultipleTabs(t *testing.T) {
	b := newBrowserLifecycle(30 * time.Millisecond)
	defer b.close()
	time.Sleep(40 * time.Millisecond)
	assertBrowserOpen(t, b) // no browser has connected yet
	first, ok := b.attach()
	if !ok {
		t.Fatal("first attachment refused")
	}
	second, _ := b.attach()
	first()
	first() // duplicate cleanup must be harmless
	time.Sleep(40 * time.Millisecond)
	assertBrowserOpen(t, b) // another tab is still open
	second()
	reload, ok := b.attach()
	if !ok {
		t.Fatal("reload refused")
	}
	time.Sleep(40 * time.Millisecond)
	assertBrowserOpen(t, b)
	reload()
	select {
	case <-b.done:
	case <-time.After(time.Second):
		t.Fatal("last tab did not stop the application")
	}
	if _, ok = b.attach(); ok {
		t.Fatal("attachment accepted after shutdown")
	}
}
func TestPresenceConnectionTracksBrowserAndRequiresLocalToken(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := New(db, "unused")
	s.TrackBrowser()
	s.browser.grace = 30 * time.Millisecond
	defer s.Close()
	remote := httptest.NewUnstartedServer(nil)
	s.Host = remote.Listener.Addr().String()
	remote.Config.Handler = s.Handler()
	remote.Start()
	defer remote.Close()
	response, e := remote.Client().Post(remote.URL+"/api/browser/presence", "text/plain", nil)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal(response.StatusCode)
	}
	assertBrowserOpen(t, s.browser)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, _ := http.NewRequestWithContext(ctx, "POST", remote.URL+"/api/browser/presence", nil)
	r.Header.Set("X-CSRF-Token", s.token)
	response, e = remote.Client().Do(r)
	if e != nil {
		t.Fatal(e)
	}
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal(response.StatusCode)
	}
	time.Sleep(50 * time.Millisecond)
	assertBrowserOpen(t, s.browser) // not an inactivity timeout
	cancel()
	response.Body.Close()
	select {
	case <-s.browser.done:
	case <-time.After(time.Second):
		t.Fatal("closed HTTP connection did not end browser lifetime")
	}
}
