package cli

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRisolviScheda(t *testing.T) {
	cases := []struct {
		in, want string
		wantErr  bool
	}{
		{"", "", false},
		{"7", "7", false},
		{"esiti", "7", false},
		{"8a", "8a", false},
		{"AD3", "", true},
		{"P7_1_1", "", true},
		{"7,8a", "", true},
	}
	for _, c := range cases {
		got, err := risolviScheda(c.in)
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("risolviScheda(%q) = %q, err=%v; want %q, err=%v", c.in, got, err, c.want, c.wantErr)
		}
	}
}

func TestVerificaRicercaEsatta(t *testing.T) {
	if verificaRicercaEsatta(true, "") != nil {
		t.Error("ricerca estesa senza tipologia: nessun errore atteso")
	}
	if verificaRicercaEsatta(false, "7") != nil {
		t.Error("ricerca esatta con tipologia: nessun errore atteso")
	}
	if verificaRicercaEsatta(false, "") == nil {
		t.Error("ricerca esatta senza tipologia: errore atteso")
	}
}

func TestPaginatedGetConservaFuzzyFalse(t *testing.T) {
	fc := &paramsCatcher{}
	_, _ = paginatedGet(t.Context(), fc, "/avvisi-full-text", map[string]string{
		"atlasFuzzySearchEnabled": "false",
		"ricercaArchivio":         "false",
		"page":                    "0",
	}, nil, false, "", "offset", "", "", "")
	if fc.params["atlasFuzzySearchEnabled"] != "false" {
		t.Errorf("atlasFuzzySearchEnabled=false deve arrivare al servizio, params=%v", fc.params)
	}
	if _, ok := fc.params["ricercaArchivio"]; ok {
		t.Errorf("gli altri false restano esclusi, params=%v", fc.params)
	}
}

type paramsCatcher struct{ params map[string]string }

func (p *paramsCatcher) GetWithHeaders(_ context.Context, _ string, params map[string]string, _ map[string]string) (json.RawMessage, error) {
	p.params = params
	return json.RawMessage(`{"content":[]}`), nil
}
