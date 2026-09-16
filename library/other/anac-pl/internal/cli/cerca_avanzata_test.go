package cli

import "testing"

func TestValidateCPVFilter(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", false},
		{"30213000", false},
		{"302", false},
		{"45", true},     // dal 15/09/2026 il servizio vuole almeno 3 cifre
		{"72,302", true}, // anche dentro una lista
		{"722", false},
		{"30213000-5", false},           // forma degli atti ufficiali
		{"30213000-9", false},           // il servizio ignora la cifra, giusta o sbagliata
		{"30213000-5, 42120000", false}, // più valori, con e senza cifra
		{"302-5", true},                 // prefisso + cifra: il servizio non trova nulla
		{"30213000-", true},             // trattino senza cifra
		{"30213000-55", true},           // due cifre dopo il trattino
		{"3021300-5", true},             // base di 7 cifre
		{"Computer personali", true},    // descrizione
		{"3", true},                     // meno di 3 cifre
		{"302130001", true},             // più di 8 cifre senza trattino
	}
	for _, c := range cases {
		err := validateCPVFilter("--cpv", c.in, 3)
		if (err != nil) != c.wantErr {
			t.Errorf("validateCPVFilter(%q) err=%v; wantErr=%v", c.in, err, c.wantErr)
		}
	}
}

func TestCodiciCPVESoloCodiciCompleti(t *testing.T) {
	if got := codiciCPV(" 30213000-5, 42120000 ,302"); len(got) != 3 || got[0] != "30213000" || got[1] != "42120000" || got[2] != "302" {
		t.Errorf("codiciCPV = %q", got)
	}
	for in, want := range map[string]bool{
		"30213000":            true,
		"30213000-5":          true, // col trattino resta un codice completo
		"30213000-5,42120000": true,
		"302":                 false,
		"30213000,302":        false,
		"":                    false,
	} {
		if got := soloCodiciCompleti(in); got != want {
			t.Errorf("soloCodiciCompleti(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestValidateCPVFilterMinimoPerFlag(t *testing.T) {
	if validateCPVFilter("--cpv-exact", "72", 2) != nil {
		t.Error("--cpv-exact filtra in locale: la divisione a 2 cifre resta ammessa")
	}
	if validateCPVFilter("--cpv-code", "72", 3) == nil {
		t.Error("--cpv-code arriva al servizio, che rifiuta meno di 3 cifre")
	}
}
