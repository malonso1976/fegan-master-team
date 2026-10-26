package fegan

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestChampionshipLinksExcludeOtherMeetsAndComments(t *testing.T) {
	b := []byte(`<table><!-- <tr><td>2025-06-01 Cto. Galego Master de Veran</td><td><a href="old.pdf">PDF</a></td></tr> --><tr><td>2026-06-13</td><td>Cto. Galego Máster de Verán</td><td><a href="result.pdf">PDF</a><a href="archive.zip">ZIP</a><a href="web/index.html">WEB</a></td></tr><tr><td>2026-02-15 Cto. Galego Máster de Fondo</td><td><a href="fondo.pdf">PDF</a></td></tr></table>`)
	got := championships(b, "https://www.fegan.org/fegan/tprd/2526/index.htm")
	if len(got) != 1 || got[0].Date != "2026-06-13" || got[0].SeasonStart != 2025 || got[0].ZIP != "https://www.fegan.org/fegan/tprd/2526/archive.zip" {
		t.Fatal(got)
	}
	if absolute("https://www.fegan.org/post/20", "http://www.fegan.org/fegan/tprd/2425/index.htm") != "https://www.fegan.org/fegan/tprd/2425/index.htm" {
		t.Fatal("http not upgraded")
	}
}
func TestAliasesReadOfficialClubCodesFromCompressedLenex(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	f, e := writer.Create("results.lef")
	if e != nil {
		t.Fatal(e)
	}
	f.Write([]byte(`<LENEX><CLUB code="00123" name="Nombre largo" shortname="ALIAS"/></LENEX>`))
	writer.Close()
	aliases, e := lenexAliases(buffer.Bytes())
	if e != nil || aliases["ALIAS"] != "00123" {
		t.Fatal(aliases, e)
	}
}
