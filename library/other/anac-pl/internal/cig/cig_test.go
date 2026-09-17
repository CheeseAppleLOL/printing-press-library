package cig

import "testing"

func TestVerificaCIGReali(t *testing.T) {
	cases := []struct {
		cig, tipo string
		valido    bool
	}{
		{"5527244A08", Simog, true},
		{"5527244A09", Simog, false},
		{"B057DC558A", SimogPCP, true},
		{"b7e26b1dc7", SimogPCP, true}, // minuscole
		{"B7E26B1DC8", SimogPCP, false},
		{"Z94375BBCC", SmartCIG, true},
		{"Z94375BBCD", SmartCIG, false},
		{"Z6C375BBCC", SmartCIG, false},
		{"B7E26B1DC", "", false},  // 9 caratteri
		{"V7E26B1DC7", "", false}, // iniziale fuori dalle famiglie
	}
	for _, c := range cases {
		e := Verifica(c.cig)
		if e.Valido != c.valido || e.Tipo != c.tipo {
			t.Errorf("Verifica(%q) = valido %v tipo %q (%s); atteso valido %v tipo %q", c.cig, e.Valido, e.Tipo, e.Motivo, c.valido, c.tipo)
		}
		if !e.Valido && e.Motivo == "" {
			t.Errorf("Verifica(%q): un CIG non valido deve dire perché", c.cig)
		}
	}
}

func TestSembraCIG(t *testing.T) {
	for s, want := range map[string]bool{
		"B7E26B1DC8": true, "5527244A08": true, "Z94375BBCC": true,
		"microsoft": false, "MANUTENZIO": false, "abbaccadde": false, "B7E26B1DC": false, "servizi di": false,
	} {
		if got := SembraCIG(s); got != want {
			t.Errorf("SembraCIG(%q) = %v; want %v", s, got, want)
		}
	}
}
