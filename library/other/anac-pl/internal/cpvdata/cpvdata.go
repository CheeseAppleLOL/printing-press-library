// Package cpvdata embeds the EU Common Procurement Vocabulary (CPV 2008)
// with official Italian descriptions, so the CLI can browse and search CPV
// codes fully offline. Source: official EU TED CPV 2008 vocabulary
// (Reg. CE 213/2008), Italian column.
package cpvdata

import (
	_ "embed"
	"sort"
	"strings"
)

//go:embed cpv_it.tsv
var raw string

// Entry is a single CPV code with its Italian description.
type Entry struct {
	Code        string `json:"code"`        // 8-digit CPV code, e.g. "72000000"
	Description string `json:"description"` // official Italian description
	Level       int    `json:"level"`       // 1=division ... 5=most specific
}

var entries []Entry
var byCode map[string]Entry
var byDesc map[string]string // descrizione normalizzata -> codice

func init() {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	entries = make([]Entry, 0, len(lines))
	byCode = make(map[string]Entry, len(lines))
	byDesc = make(map[string]string, len(lines))
	for _, ln := range lines {
		parts := strings.SplitN(ln, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		e := Entry{Code: parts[0], Description: parts[1], Level: cpvLevel(parts[0])}
		entries = append(entries, e)
		byCode[e.Code] = e
		byDesc[normDesc(parts[1])] = parts[0]
	}
}

func normDesc(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// NormalizeCPV ricava un codice CPV a 8 cifre da un valore eterogeneo del campo
// `cpv` ANAC: oggetto {codice,descrizione}, stringa col codice, o stringa con la
// sola descrizione (mappata al codice via vocabolario). Ritorna (codice, descrizione, ok).
func NormalizeCPV(v any) (string, string, bool) {
	switch x := v.(type) {
	case map[string]any:
		code, _ := x["codice"].(string)
		desc, _ := x["descrizione"].(string)
		if i := strings.IndexByte(code, '-'); i >= 0 {
			code = code[:i] // cifra di controllo, come nel caso stringa
		}
		if code == "" && desc != "" {
			if c, ok := byDesc[normDesc(desc)]; ok {
				code = c
			}
		}
		if code != "" {
			if e, ok := byCode[code]; ok && desc == "" {
				desc = e.Description
			}
			return code, desc, true
		}
		return "", desc, desc != ""
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", "", false
		}
		// "72412000_Fornitori di servizi di posta elettronica": dal settembre
		// 2026 il dettaglio dell'avviso espone codice e descrizione in un solo
		// valore, separati dal primo trattino basso.
		if i := strings.IndexByte(s, '_'); i > 0 {
			code := s[:i]
			if j := strings.IndexByte(code, '-'); j >= 0 {
				code = code[:j]
			}
			if allDigits(code) {
				desc := strings.TrimSpace(s[i+1:])
				if e, ok := byCode[code]; ok && desc == "" {
					desc = e.Description
				}
				return code, desc, true
			}
		}
		// stringa numerica = codice (eventuale "-N" check digit rimosso)
		base := s
		if i := strings.IndexByte(base, '-'); i >= 0 {
			base = base[:i]
		}
		if allDigits(base) {
			if e, ok := byCode[base]; ok {
				return base, e.Description, true
			}
			return base, "", true
		}
		// altrimenti è una descrizione: mappa al codice
		if c, ok := byDesc[normDesc(s)]; ok {
			return c, s, true
		}
		return "", s, true // descrizione senza codice noto
	}
	return "", "", false
}

// cpvLevel derives the hierarchy depth from the code's trailing-zero structure.
// Division XX000000 -> 1, then each further non-zero group deepens the level.
func cpvLevel(code string) int {
	if len(code) < 8 {
		return 0
	}
	// division (2 digits) is level 1; digits 3..8 add depth as they become non-zero
	level := 1
	tail := code[2:]
	for i := 0; i < len(tail); i++ {
		if tail[i] != '0' {
			level++
		}
	}
	if level > 5 {
		level = 5
	}
	return level
}

// Count returns the number of CPV entries.
func Count() int { return len(entries) }

// Get returns the entry for an exact code. The code may be passed with or
// without the "-N" check digit; only the 8-digit base is matched.
func Get(code string) (Entry, bool) {
	code = strings.TrimSpace(code)
	if strings.Contains(code, "-") {
		c, ok := CodiceConControllo(code)
		if !ok {
			return Entry{}, false
		}
		code = c
	}
	e, ok := byCode[code]
	return e, ok
}

// CodiceConControllo riconosce la forma ufficiale del codice con la cifra di
// controllo, esattamente 8 cifre, trattino e una cifra (30213000-5), e
// restituisce il codice a 8 cifre. Varianti come 30213000-55 o 302-5 non sono
// codici: accettarle troncando il suffisso trasformerebbe un errore in una
// ricerca più ampia di quella chiesta.
func CodiceConControllo(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) != 10 || s[8] != '-' || !allDigits(s[:8]) || !allDigits(s[9:]) {
		return "", false
	}
	return s[:8], true
}

// Search returns entries matching the query. A purely numeric query matches by
// code prefix; otherwise every whitespace-separated token must appear
// (case-insensitively) in the description or code. Results are ordered by code.
// limit <= 0 returns all matches.
func Search(query string, limit int) []Entry {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	// Un codice con la cifra di controllo (30213000-5) è ancora una ricerca
	// per codice: senza questo passaggio finiva fra le parole e non trovava nulla.
	if c, ok := CodiceConControllo(query); ok {
		query = c
	}
	tokens := strings.Fields(strings.ToLower(query))
	numericPrefix := allDigits(query)

	var out []Entry
	for _, e := range entries {
		if numericPrefix {
			if strings.HasPrefix(e.Code, query) {
				out = append(out, e)
			}
			continue
		}
		hay := strings.ToLower(e.Description + " " + e.Code)
		match := true
		for _, t := range tokens {
			if !strings.Contains(hay, t) {
				match = false
				break
			}
		}
		if match {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
