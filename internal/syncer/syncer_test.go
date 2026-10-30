package syncer

import (
	"context"
	"fmt"
	"github.com/malonso1976/fegan-master-team/internal/domain"
	"github.com/malonso1976/fegan-master-team/internal/fegan"
	"github.com/malonso1976/fegan-master-team/internal/store"
	"io"
	"net/http"
	"os"
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
	if status.Running || status.Completed != 4 || status.Total != 4 || len(status.Warnings) > 0 || status.FinishedAt == "" {
		t.Fatal(status)
	}
	var persisted Status
	if e = db.Setting(ctx, "sync_status", &persisted); e != nil || persisted.FinishedAt == "" {
		t.Fatal(persisted, e)
	}
	service.Start("00123", "rivals", false)
	service.wg.Wait()
	status = service.Status()
	if requests != 4 || status.Reused != 4 {
		t.Fatal("successful daily query was repeated", requests, status)
	}
	service.Close()
}

func TestInitialClubQueriesCoverFiveYearsAndAllIndividualEvents(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	queries := clubHistoryQueries("00123", now)
	seen := map[string]bool{}
	events := 0
	for _, event := range domain.Events {
		if !domain.Relay(event) {
			events++
		}
	}
	if len(queries) != events*8 {
		t.Fatal("missing gender/pool/timing combinations")
	}
	for _, q := range queries {
		if q.From != "2021-10-27" || q.To != "2026-10-27" || q.ClubID != "00123" || !q.Member || q.License != "" || domain.Relay(q.Event) {
			t.Fatal("wrong initial import scope")
		}
		if err := q.Validate(); err != nil {
			t.Fatal(err)
		}
		key := q.Gender + q.Chrono + string(rune(q.Pool)) + q.Event
		if seen[key] {
			t.Fatal("duplicate club query")
		}
		seen[key] = true
	}
}

func TestClubHistoryImportsOldSwimmersAndPreservesExistingProfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, id := range []string{"00123", "00999"} {
		if err = db.SaveClub(ctx, domain.Club{ID: id, Name: "Club de prueba"}); err != nil {
			t.Fatal(err)
		}
	}
	existing := domain.Swimmer{License: "00001", Name: "Ficha local", BirthYear: 1986, Gender: "M", ClubID: "00999", Member: true, Available: false}
	if err = db.SaveSwimmer(ctx, existing); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../fegan/testdata/results.html")
	if err != nil {
		t.Fatal(err)
	}
	html := strings.ReplaceAll(string(fixture), "20260613", "20230613")
	start := strings.Index(html, "<tr><td>")
	end := start + strings.Index(html[start:], "</tr>") + len("</tr>")
	row := strings.ReplaceAll(strings.ReplaceAll(html[start:end], "00001", "00002"), "20230613", "20220613")
	html = strings.Replace(html, "</tbody>", row+"</tbody>", 1)
	calls := 0
	client := fegan.New()
	client.HTTP = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			cancel()
			return nil, context.Canceled
		}
		r.ParseForm()
		for key, want := range map[string]string{"combo_club": "00123", "input_from_date": "2021-10-27", "input_to_date": "2026-10-27", "input_license": ""} {
			if r.Form.Get(key) != want {
				t.Errorf("wrong initial filter %s", key)
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(html)), Header: http.Header{}}, nil
	})}
	service := New(db, client)
	defer service.Close()
	if service.loadClubHistory(ctx, "00123", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("cancellation ignored")
	}
	d, err := db.Snapshot(context.Background())
	if err != nil || len(d.Swimmers) != 2 || len(d.Marks) != 2 {
		t.Fatal("old results were not imported", err)
	}
	for _, s := range d.Swimmers {
		if s.License == existing.License && s != existing {
			t.Fatal("local profile overwritten")
		}
		if s.License == "00002" && (s.ClubID != "00123" || !s.Member) {
			t.Fatal("historical club swimmer missing")
		}
	}
	for _, m := range d.Marks {
		if m.ClubID != "00123" || m.Date >= "2024-10-27" {
			t.Fatal("history or original club lost")
		}
	}
}

func TestDailyClubCacheRequiresFiveYearCoverage(t *testing.T) {
	for _, years := range []int{0, 2, 10, 5} {
		t.Run(fmt.Sprint(years), func(t *testing.T) {
			db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err = db.SetSetting(context.Background(), "full_sync_done", fullSyncStamp{ClubID: "00123", Date: time.Now().Format("2006-01-02"), HistoryYears: years}); err != nil {
				t.Fatal(err)
			}
			client := fegan.New()
			client.HTTP = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) { return nil, context.Canceled })}
			service := New(db, client)
			status := service.Start("00123", "all", false)
			service.Suspend()
			if status.Running != (years != 5) {
				t.Fatal("obsolete daily cache reused")
			}
		})
	}
}

func TestPersonalBestsSupplementKeepsCurrentProfileAndAvailability(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "team.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, id := range []string{"00123", "00999"} {
		if err = db.SaveClub(ctx, domain.Club{ID: id, Name: "Club"}); err != nil {
			t.Fatal(err)
		}
	}
	swimmer := domain.Swimmer{License: "00001", Name: "Ficha local", BirthYear: 1986, Gender: "M", ClubID: "00999", Member: true, Available: false}
	if err = db.SaveSwimmer(ctx, swimmer); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../fegan/testdata/personal.html")
	if err != nil {
		t.Fatal(err)
	}
	client := fegan.New()
	posts := 0
	client.HTTP = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		body := `<select name="combo_club"><option value="00123">CLUBTEST</option></select>`
		if r.Method == "POST" {
			posts++
			r.ParseForm()
			if r.URL.Path != "/marks/best_results_person" || r.Form.Get("input_from_date") != "2021-10-28" || r.Form.Get("input_license") != "00001" || r.Form.Get("combo_club") != "" {
				t.Error("wrong personal supplementation scope")
			}
			body = strings.ReplaceAll(string(fixture), "E50", r.Form.Get("combo_chrono")+r.Form.Get("combo_pool"))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	service := New(db, client)
	defer service.Close()
	if !service.loadPersonalBests(ctx, "00999", time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("personal stage failed")
	}
	d, err := db.Snapshot(ctx)
	if err != nil || len(d.Marks) != 4 || len(d.Swimmers) != 1 || d.Swimmers[0] != swimmer || posts != 4 {
		t.Fatal("personal complement changed current profile", err)
	}
	for _, m := range d.Marks {
		if m.ClubID != "00123" || !strings.HasSuffix(m.SourceURL, "/marks/best_results_person") {
			t.Fatal("historical club or provenance lost")
		}
	}
	status := service.Status()
	if status.Completed != 1 || len(status.Warnings) != 0 {
		t.Fatal("personal stage not complete")
	}
}
