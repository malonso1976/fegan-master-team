package fegan

import (
	"context"
	"os"
	"testing"
)

func TestLocalOfficialPDFs(t *testing.T) {
	if os.Getenv("FEGAN_LOCAL_PDF_CHECK") != "1" {
		t.Skip("opt-in verification using downloaded public PDFs")
	}
	ctx := context.Background()
	b, e := os.ReadFile("/tmp/fegan-clubs2627.pdf")
	if e != nil {
		t.Fatal(e)
	}
	catalog, e := ParseClubsPDF(ctx, b, "https://www.fegan.org/fegan/tprd/2627/xeral/clubs_activos.pdf")
	if e != nil {
		t.Fatal(e)
	}
	if len(catalog.Clubs) != 58 {
		t.Fatal("club count", len(catalog.Clubs))
	}
	for _, c := range catalog.Clubs {
		if c.ID == "01295" && c.ShortName != "CIDSANTI" {
			t.Fatalf("bad columns %+v", c)
		}
	}
	b, e = os.ReadFile("/tmp/fegan-relay-event.pdf")
	if e != nil {
		t.Fatal(e)
	}
	r, e := ParseRelayPDF(ctx, b, "https://www.fegan.org/example.pdf", "2026-06-13", "Example", catalog.Clubs, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("parsed relay marks", len(r.Data.Marks), "warnings", len(r.Warnings))
	if len(r.Data.Marks) < 20 {
		t.Fatal("not enough parsed marks")
	}
}

func TestSyntheticPDFCatalogAndRelays(t *testing.T) {
	ctx := context.Background()
	b, e := os.ReadFile("testdata/catalog.pdf")
	if e != nil {
		t.Fatal(e)
	}
	catalog, e := ParseClubsPDF(ctx, b, "https://www.fegan.org/catalog.pdf")
	if e != nil || len(catalog.Clubs) != 2 {
		t.Fatal(catalog, e)
	}
	if catalog.Clubs[0].ID != "00123" || catalog.Clubs[0].MediumName != "CLUB PRUEBA" || catalog.Clubs[0].ShortName != "PRUEBA" {
		t.Fatal(catalog.Clubs)
	}
	b, _ = os.ReadFile("testdata/relays.pdf")
	result, e := ParseRelayPDF(ctx, b, "https://www.fegan.org/result.pdf", "2025-01-12", "Test", catalog.Clubs, nil)
	if e != nil || len(result.Data.Marks) != 1 || len(result.Warnings) != 1 {
		t.Fatal(result, e)
	}
	m := result.Data.Marks[0]
	if m.ClubID != "00123" || m.RelayCategory != 120 || m.Gender != "X" || m.Event != "4x50S" || m.Centiseconds != 12345 || m.Source != "fegan-pdf" {
		t.Fatal(m)
	}
	if _, e = ParseClubsPDF(ctx, []byte("<html>Error</html>"), ""); e == nil {
		t.Fatal("invalid PDF accepted")
	}
}
