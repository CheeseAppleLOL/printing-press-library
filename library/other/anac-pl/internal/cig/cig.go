// Package cig verifica la cifra di controllo del Codice Identificativo di Gara
// con l'algoritmo pubblicato da ANAC in anticorruzione/npa
// (docs/Algoritmo validazione CIG/algoritmoValidazioneCIG.md). Le tre famiglie
// sono verificate su CIG reali: 5527244A08 (Simog), 1.939 CIG PCP estratti da
// PVL (es. B057DC558A), Z94375BBCC (SmartCIG).
package cig

import (
	"fmt"
	"strconv"
	"strings"
)

// Famiglie di CIG.
const (
	Simog    = "SIMOG"     // ANNNNNNKKK, A cifra, N decimale
	SimogPCP = "SIMOG_PCP" // ANNNNNNKKK, A lettera A-U, N esadecimale (Simog seconda versione e PCP)
	SmartCIG = "SMARTCIG"  // AKKCCCCCCC, A in X Y Z
)

// Esito del controllo di un CIG.
type Esito struct {
	CIG    string `json:"cig"`
	Valido bool   `json:"valido"`
	Tipo   string `json:"tipo,omitempty"`
	Motivo string `json:"motivo,omitempty"`
}

// Verifica controlla struttura e cifra di controllo di un CIG, senza
// distinguere maiuscole e minuscole.
func Verifica(s string) Esito {
	c := strings.ToUpper(strings.TrimSpace(s))
	e := Esito{CIG: c}
	if len(c) != 10 {
		e.Motivo = fmt.Sprintf("un CIG ha 10 caratteri, questo ne ha %d", len(c))
		return e
	}
	a := c[0]
	switch {
	case a >= '0' && a <= '9':
		e.Tipo = Simog
		n, err := strconv.ParseUint(c[:7], 10, 64)
		if err != nil || !esadecimale(c[7:]) {
			e.Motivo = "struttura Simog: 7 cifre decimali seguite da 3 caratteri esadecimali"
			return e
		}
		if c[1:7] == "000000" {
			e.Motivo = "progressivo nullo"
			return e
		}
		return controllo(e, hex(n*211%4091, 3), c[7:])
	case a >= 'A' && a <= 'U':
		e.Tipo = SimogPCP
		n, err := strconv.ParseUint(c[1:7], 16, 64)
		if err != nil || !esadecimale(c[7:]) {
			e.Motivo = "struttura Simog/PCP: una lettera da A a U, 6 caratteri esadecimali e 3 di controllo"
			return e
		}
		return controllo(e, hex((n+uint64(a-'A'+1))*211%4091, 3), c[7:])
	case a == 'X' || a == 'Y' || a == 'Z':
		e.Tipo = SmartCIG
		n, err := strconv.ParseUint(c[3:], 16, 64)
		if err != nil || !esadecimale(c[1:3]) {
			e.Motivo = "struttura SmartCIG: X, Y o Z, 2 caratteri di controllo e 7 esadecimali"
			return e
		}
		if n == 0 {
			e.Motivo = "progressivo nullo"
			return e
		}
		return controllo(e, hex(n*211%251, 2), c[1:3])
	}
	e.Motivo = "il primo carattere deve essere una cifra, una lettera da A a U, oppure X, Y o Z"
	return e
}

// SembraCIG dice se una stringa ha la forma di un CIG di una delle tre
// famiglie, a prescindere dalla cifra di controllo. Serve a riconoscere un CIG
// dentro un testo libero senza scambiarlo per una parola: una parola italiana
// di 10 lettere quasi mai è fatta di sole cifre esadecimali dopo l'iniziale.
func SembraCIG(s string) bool {
	c := strings.ToUpper(strings.TrimSpace(s))
	if len(c) != 10 {
		return false
	}
	switch a := c[0]; {
	case a >= '0' && a <= '9':
		return decimale(c[:7]) && esadecimale(c[7:])
	case a >= 'A' && a <= 'U':
		return esadecimale(c[1:]) && strings.ContainsAny(c[1:], "0123456789")
	case a == 'X' || a == 'Y' || a == 'Z':
		return esadecimale(c[1:]) && strings.ContainsAny(c[1:], "0123456789")
	}
	return false
}

func controllo(e Esito, atteso, trovato string) Esito {
	if atteso != trovato {
		e.Motivo = fmt.Sprintf("cifra di controllo errata: attesa %s, trovata %s", atteso, trovato)
		return e
	}
	e.Valido = true
	return e
}

func hex(n uint64, width int) string {
	s := strings.ToUpper(strconv.FormatUint(n, 16))
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func esadecimale(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return s != ""
}

func decimale(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
