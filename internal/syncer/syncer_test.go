package syncer

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/domain"
	"github.com/marianoao/fegan-master-team/internal/fegan"
	"github.com/marianoao/fegan-master-team/internal/store"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestBackgroundRivalsCompleteAndPersistStatus(t *testing.T) {
	ctx := context.Background()
	db, e := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SaveClub(ctx, domain.Club{ID: "00123", Name: "Club"})
	db.SaveSwimmer(ctx, domain.Swimmer{License: "00001", Name: "Test", BirthYear: 1986, Gender: "M", ClubID: "00123", Member: true})
	db.SaveMark(ctx, domain.Mark{License: "00001", ClubID: "00123", Event: "50L", Pool: 25, Chrono: "E", Date: time.Now().AddDate(0, 0, -1).Format("2006-01-02"), Centiseconds: 3000, Source: "manual", Gender: "M"})
	requests := 0
	client := fegan.New()
	client.HTTP = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		requests++
		r.ParseForm()
		if r.Form.Get("combo_show_first") != "10" || r.Form.Get("input_best_mark") != "on" {
			t.Error("top-ten filters missing")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<select name="combo_club"></select>Non se atopou ningún resultado.`)), Header: http.Header{}}, nil
	})}
	service := New(db, client)
	status := service.Start("00123", "rivals", true)
	if !status.Running {
		t.Fatal(status)
	}
	service.wg.Wait()
	status = service.Status()
	if status.Running || status.Completed != 1 || status.Total != 1 || len(status.Warnings) > 0 || status.FinishedAt == "" {
		t.Fatal(status)
	}
	var persisted Status
	if e = db.Setting(ctx, "sync_status", &persisted); e != nil || persisted.FinishedAt == "" {
		t.Fatal(persisted, e)
	}
	service.Start("00123", "rivals", false)
	service.wg.Wait()
	status = service.Status()
	if requests != 1 || status.Reused != 1 {
		t.Fatal("successful daily query was repeated", requests, status)
	}
	service.Close()
}
