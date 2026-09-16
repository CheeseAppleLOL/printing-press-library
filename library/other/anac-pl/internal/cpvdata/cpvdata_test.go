// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cpvdata

import (
	"strings"
	"testing"
)

func TestVocabolarioCaricato(t *testing.T) {
	if Count() < 9000 {
		t.Fatalf("vocabolario CPV incompleto: %d voci", Count())
	}
	e, ok := Get("30213000")
	if !ok {
		t.Fatal("codice 30213000 assente dal vocabolario")
	}
	if e.Description == "" {
		t.Error("la voce deve portare la descrizione italiana")
	}
}

// Il 16/09/2026 il vocabolario aveva a 03116200 la descrizione di 03117140, e
// 03117140 mancava: confrontato con l'elenco CPV pubblicato da ANAC
// (anticorruzione/npa, tipologiche/CPV.json, 9.454 voci) e con il CPV 2008.
func TestVocabolarioAllineatoAllElencoANAC(t *testing.T) {
	if Count() != 9454 {
		t.Errorf("voci: %d, attese 9454 come nell'elenco ANAC", Count())
	}
	for code, want := range map[string]string{
		"03116200": "Lattice naturale",
		"03117140": "Piante utilizzate per la preparazione di fungicidi o simili",
	} {
		if e, ok := Get(code); !ok || e.Description != want {
			t.Errorf("%s = %q (trovato=%v); atteso %q", code, e.Description, ok, want)
		}
	}
}

func TestNormalizeCPV(t *testing.T) {
	cases := []struct {
		name     string
		in       any
		wantCode string
		wantOK   bool
	}{
		{"codice come stringa", "30213000", "30213000", true},
		{"codice con cifra di controllo", "30213000-5", "30213000", true},
		{"oggetto con codice", map[string]any{"codice": "30213000", "descrizione": ""}, "30213000", true},
		{"codice_descrizione (dettaglio avviso da settembre 2026)", "72412000_Fornitori di servizi di posta elettronica", "72412000", true},
		{"codice con cifra di controllo e descrizione", "72412000-9_Fornitori di servizi di posta elettronica", "72412000", true},
		{"descrizione con trattino basso, senza codice", "servizi_vari", "", true},
		{"oggetto con cifra di controllo", map[string]any{"codice": "30213000-5", "descrizione": ""}, "30213000", true},
		{"stringa vuota", "", "", false},
		{"tipo non gestito", 42, "", false},
	}
	for _, c := range cases {
		code, _, ok := NormalizeCPV(c.in)
		if code != c.wantCode || ok != c.wantOK {
			t.Errorf("%s: NormalizeCPV(%v) = (%q, %v), atteso (%q, %v)", c.name, c.in, code, ok, c.wantCode, c.wantOK)
		}
	}
}

func TestNormalizeCPVRisaleAlCodiceDallaDescrizione(t *testing.T) {
	e, ok := Get("30213000")
	if !ok {
		t.Skip("codice di riferimento assente")
	}
	code, _, ok := NormalizeCPV(e.Description)
	if !ok || code != "30213000" {
		t.Errorf("dalla descrizione %q atteso il codice 30213000, ottenuto %q (ok=%v)", e.Description, code, ok)
	}
}

func TestSearchTrovaPerParola(t *testing.T) {
	res := Search("posta elettronica", 10)
	if len(res) == 0 {
		t.Fatal("nessun risultato per \"posta elettronica\"")
	}
	if len(res) > 10 {
		t.Errorf("limit non rispettato: %d risultati", len(res))
	}
}

func TestSearchConCifraDiControllo(t *testing.T) {
	a := Search("30213000", 0)
	b := Search("30213000-5", 0)
	if len(a) == 0 || len(a) != len(b) || a[0].Code != b[0].Code {
		t.Errorf("Search con cifra di controllo: %v; senza: %v", b, a)
	}
}

func TestCodiceConControlloRigoroso(t *testing.T) {
	for in, want := range map[string]bool{
		"30213000-5": true, " 30213000-5 ": true,
		"30213000-55": false, "302-5": false, "30213000-": false, "30213000-a": false, "3021300-5": false, "30213000": false,
	} {
		if _, ok := CodiceConControllo(in); ok != want {
			t.Errorf("CodiceConControllo(%q) ok=%v; want %v", in, ok, want)
		}
	}
	if _, ok := Get("30213000-5"); !ok {
		t.Error("Get(30213000-5): atteso trovato")
	}
	for _, in := range []string{"30213000-55", "30213000-abc", "302-5"} {
		if _, ok := Get(in); ok {
			t.Errorf("Get(%q): atteso non trovato", in)
		}
		for _, e := range Search(in, 0) {
			if strings.HasPrefix(e.Code, "302") {
				t.Errorf("Search(%q) non deve allargarsi ai codici 302…: trovato %s", in, e.Code)
				break
			}
		}
	}
}
