package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestWarnCIGNonValido(t *testing.T) {
	casi := []struct {
		query  string
		avvisa bool
	}{
		{"B7E26B1DC8", true},
		{"B7E26B1DC8,", true},
		{"(B7E26B1DC8)", true},
		{"CIG:B7E26B1DC8", true},
		{"lavori cig=B7E26B1DC8 scuola", true},
		{"B7E26B1DC7", false},
		{"CIG:B7E26B1DC7", false},
		{"microsoft office", false},
		{"MANUTENZIO", false},
	}
	for _, c := range casi {
		var buf bytes.Buffer
		warnCIGNonValido(&buf, c.query)
		if got := strings.Contains(buf.String(), "non è valido"); got != c.avvisa {
			t.Errorf("warnCIGNonValido(%q): avviso=%v, atteso %v (%q)", c.query, got, c.avvisa, buf.String())
		}
	}
}

func TestCigCheckEsitiEUscita(t *testing.T) {
	esegui := func(args ...string) (string, error) {
		cmd := newCigCheckCmd(&rootFlags{asJSON: true})
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	out, err := esegui("B7E26B1DC8", "5527244A08")
	if err != nil {
		t.Fatalf("cifra di controllo errata deve uscire 0 con l'esito, got %v", err)
	}
	if !strings.Contains(out, `"valido": false`) || !strings.Contains(out, "attesa DC7") {
		t.Errorf("esito per singolo CIG mancante: %s", out)
	}
	out, err = esegui("5527244A08", "__printing_press_invalid__")
	if err == nil || ExitCode(err) != 2 {
		t.Fatalf("argomento che non è un CIG: atteso errore d'uso exit 2, got %v", err)
	}
	if out != "" {
		t.Errorf("errore d'uso non deve stampare output parziale: %q", out)
	}
}
