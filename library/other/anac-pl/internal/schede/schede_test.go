package schede

import (
	"strings"
	"testing"
)

func TestElencoSchede(t *testing.T) {
	if n := len(Tutte()); n != 150 {
		t.Errorf("schede: %d, attese 150 come in codiceScheda.json", n)
	}
	for _, c := range []string{"AD3", "a1_29", "P1_16", "NAG", "M1"} {
		s, ok := Get(c)
		if !ok || s.Descrizione == "" {
			t.Errorf("Get(%q): trovato=%v descrizione=%q", c, ok, s.Descrizione)
		}
	}
	if s, _ := Get("AD3"); !strings.HasPrefix(s.Descrizione, "Affidamento diretto") {
		t.Errorf("AD3 = %q", s.Descrizione)
	}
	if _, ok := Get("XYZ"); ok {
		t.Error("codice inesistente trovato")
	}
}
