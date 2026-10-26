package fegan

import (
	"context"
	"github.com/marianoao/fegan-master-team/internal/store"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveDocumentSources(t *testing.T) {
	if os.Getenv("FEGAN_LIVE_DOCUMENT_CHECK") != "1" {
		t.Skip("opt-in public-source integration check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c := New()
	catalog, e := c.ClubCatalog(ctx, 2026)
	if e != nil {
		t.Fatal(e)
	}
	championships, e := c.Championships(ctx, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if len(championships) != 4 {
		t.Fatalf("want four championships got %d", len(championships))
	}
	db, e := store.Open(filepath.Join(t.TempDir(), "pdf.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, ch := range championships {
		t.Log("source", ch.Date)
		result, e := c.RelayChampionship(ctx, ch, catalog.Clubs)
		if e != nil {
			t.Fatal(e)
		}
		t.Log(ch.Date, "marks", len(result.Data.Marks), "warnings", result.Warnings)
		first, e := db.Import(ctx, result.Data)
		if e != nil || first.Inserted != len(result.Data.Marks) {
			t.Fatal(first, e)
		}
		second, e := db.Import(ctx, result.Data)
		if e != nil || second.Inserted != 0 || second.Duplicates != len(result.Data.Marks) {
			t.Fatal("duplicate import", second, e)
		}
		if len(result.Data.Marks) < 60 {
			t.Fatal("too few relays")
		}
	}
}

func TestLiveRivalFallback(t *testing.T) {
	path := os.Getenv("FEGAN_LIVE_RIVAL_DB")
	if path == "" {
		t.Skip("opt-in live rival check with temporary team data")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, e := store.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	d, e := db.Snapshot(ctx)
	if e != nil {
		t.Fatal(e)
	}
	client := New()
	tested := 0
	for _, q := range RivalQueries(d, "01295", 2026, time.Now()) {
		if q.Event != "100M" || q.Pool != 25 || q.Chrono != "M" {
			continue
		}
		p, e := client.Preview(ctx, q)
		if e != nil {
			t.Errorf("query gender=%s births=%d-%d: %s", q.Gender, q.BirthFrom, q.BirthTo, e)
		} else {
			t.Log("group", q.Gender, q.BirthFrom, q.BirthTo, "marks", len(p.Data.Marks))
		}
		tested++
	}
	if tested == 0 {
		t.Fatal("no queries checked")
	}
}
