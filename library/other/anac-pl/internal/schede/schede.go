// Package schede descrive i codici scheda degli avvisi (AD3, A1_29, P1_16...)
// con l'elenco pubblicato da ANAC in anticorruzione/npa
// (docs/modello-dati/tipologiche/codiceScheda.json, 150 voci).
package schede

import (
	_ "embed"
	"strings"
)

//go:embed schede_it.tsv
var raw string

// Scheda è un codice scheda con la sua descrizione ufficiale.
type Scheda struct {
	Codice      string `json:"codice"`
	Descrizione string `json:"descrizione"`
}

var elenco []Scheda
var perCodice map[string]Scheda

func init() {
	righe := strings.Split(strings.TrimSpace(raw), "\n")
	elenco = make([]Scheda, 0, len(righe))
	perCodice = make(map[string]Scheda, len(righe))
	for _, r := range righe {
		parti := strings.SplitN(r, "\t", 2)
		if len(parti) != 2 {
			continue
		}
		s := Scheda{Codice: parti[0], Descrizione: parti[1]}
		elenco = append(elenco, s)
		perCodice[strings.ToUpper(s.Codice)] = s
	}
}

// Get restituisce la scheda per codice, senza distinguere maiuscole e minuscole.
func Get(codice string) (Scheda, bool) {
	s, ok := perCodice[strings.ToUpper(strings.TrimSpace(codice))]
	return s, ok
}

// Tutte restituisce l'elenco completo, ordinato per codice.
func Tutte() []Scheda {
	out := make([]Scheda, len(elenco))
	copy(out, elenco)
	return out
}
