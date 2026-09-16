package cli

import (
	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cpvdata"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/client"

	"github.com/spf13/cobra"
)

// specializzataPath è l'endpoint della "Ricerca avanzata" rilasciata da ANAC in
// beta a luglio 2026. A differenza di /avvisi-full-text, il suo filtro `cpv`
// seleziona davvero per codice CPV (o per un suo prefisso) e funziona su tutto
// l'archivio. Vedi docs/verifica-beta-cpv-2026-08.md.
const specializzataPath = "/avvisi-full-text-specializzata"

// archivioStart è la data di inizio della copertura della piattaforma: gli
// avvisi pubblicati iniziano nel 2024. L'endpoint richiede sempre entrambe le
// date, quindi la usiamo come default per "tutto l'archivio".
const archivioStart = "01/01/2024"

// specializzataParams costruisce i parametri comuni della ricerca avanzata.
// Le due date sono obbligatorie lato server (senza, risponde 400).
func specializzataParams(cpv, sa, categorie, from, to string, or bool, size int) map[string]string {
	if from == "" {
		from = archivioStart
	}
	if to == "" {
		to = time.Now().Format("02/01/2006")
	}
	params := map[string]string{
		"pageSize":               strconv.Itoa(size),
		"dataPubblicazioneStart": from,
		"dataPubblicazioneEnd":   to,
		"sortField":              "dataPubblicazione",
		"sortDirection":          "asc",
		"operatore":              "AND",
	}
	if or {
		params["operatore"] = "OR"
	}
	if cpv != "" {
		params["cpv"] = cpv
	}
	if sa != "" {
		params["sa"] = sa
	}
	if categorie != "" {
		// Il parametro si chiama `lavori`, non `categorie`: passando un nome che
		// il server non conosce la richiesta va a buon fine ma senza quel filtro,
		// restituendo il totale non filtrato (verificato: 92.390 avvisi a luglio
		// 2026 con `categorie=OG 1`, contro i 2.471 di `lavori=OG 1`).
		// Il valore va scritto come lo serve /api/v0/lavori?request=visible,
		// spazio compreso: "OG 1" funziona, "OG1" non restituisce nulla.
		params["lavori"] = categorie
	}
	return params
}

// fetchSpecializzata scorre le pagine della ricerca avanzata e restituisce gli
// avvisi grezzi. Il parametro `page` è dichiarato ma ignorato dal server: la
// paginazione è a token, come nel vecchio endpoint.
func fetchSpecializzata(ctx context.Context, c *client.Client, base map[string]string, pages int) ([]json.RawMessage, int64, error) {
	var out []json.RawMessage
	seen := map[string]bool{}
	token := ""
	var total int64
	for p := 0; p < pages; p++ {
		params := map[string]string{}
		for k, v := range base {
			params[k] = v
		}
		if token != "" {
			params["direzionePaginazione"] = "AVANTI"
			params["tokenPaginazione"] = token
		}
		data, err := c.Get(ctx, specializzataPath, params)
		if err != nil {
			return out, total, err
		}
		var env struct {
			Content             []json.RawMessage `json:"content"`
			Count               float64           `json:"count"`
			LastPaginationToken string            `json:"lastPaginationToken"`
		}
		if json.Unmarshal(data, &env) != nil || len(env.Content) == 0 {
			break
		}
		if env.Count > 0 {
			total = int64(env.Count)
		}
		added := 0
		for _, raw := range env.Content {
			var idOnly struct {
				IDAvviso string `json:"idAvviso"`
			}
			_ = json.Unmarshal(raw, &idOnly)
			if idOnly.IDAvviso == "" || seen[idOnly.IDAvviso] {
				continue
			}
			seen[idOnly.IDAvviso] = true
			out = append(out, raw)
			added++
		}
		if added == 0 || env.LastPaginationToken == "" {
			break
		}
		token = env.LastPaginationToken
	}
	return out, total, nil
}

// newCercaAvanzataCmd espone la "Ricerca avanzata" di ANAC, l'unica in cui il
// filtro CPV è selettivo. Rispetto al form web non impone la finestra massima
// di 12 mesi (limite solo lato interfaccia) e pagina in automatico.
func newCercaAvanzataCmd(flags *rootFlags) *cobra.Command {
	var cpv, sa, categorie, from, to string
	var or bool
	var size, pages int

	cmd := &cobra.Command{
		Use:     "cerca-avanzata",
		Aliases: []string{"avanzata"},
		Short:   "Ricerca avanzata: filtro CPV che seleziona davvero per codice (endpoint specializzato)",
		Long: strings.Trim(`
Ricerca avvisi con il filtro CPV corretto, quello della "Ricerca avanzata" del
portale (rilasciato in beta a luglio 2026).

A differenza di 'cerca', qui il CPV è un vero filtro sul codice del lotto:
  --cpv 30213000     codice completo
  --cpv 30213000-5   codice completo con la cifra di controllo, come negli atti
  --cpv 30213        prefisso: tutta la famiglia
  --cpv 302,4512     più valori separati da virgola, in OR fra loro

Le date sono obbligatorie lato API: se non le passi vengono usate 01/01/2024
(inizio della copertura della piattaforma) e oggi. Il form web impone anche un
intervallo massimo di 12 mesi, l'API no: da qui si interroga tutto l'archivio in
un colpo solo.

Limiti dell'endpoint: non accetta ricerca testuale, tipologia avviso o importo.
Per quelli usa 'cerca' (il cui filtro CPV però non è selettivo).
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli cerca-avanzata --cpv 30213000
  anac-pl-pp-cli cerca-avanzata --cpv 72412000 --from 01/01/2024 --to 31/12/2024
  anac-pl-pp-cli cerca-avanzata --cpv 302 --pages 5 --json
  anac-pl-pp-cli cerca-avanzata --cpv 30213000 --sa 13187590156 --or
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if cpv == "" && sa == "" && categorie == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("serve almeno un filtro: --cpv, --sa o --categorie"))
			}
			if err := validateCPVFilter("--cpv", cpv, 3); err != nil {
				_ = cmd.Usage()
				return usageErr(err)
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would search avvisi via ricerca avanzata (live)")
				return nil
			}
			if size <= 0 {
				size = 20
			}
			if pages <= 0 {
				pages = 1
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			base := specializzataParams(cpv, sa, categorie, from, to, or, size)
			items, total, err := fetchSpecializzata(cmd.Context(), c, base, pages)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			warnConteggioSottostimato(cmd.ErrOrStderr(), total, size, cpv)

			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				env := map[string]any{"count": total, "content": items}
				return printJSONFiltered(cmd.OutOrStdout(), env, flags)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "risultati totali: %d (scaricati %d, %d pagine da %d)\n", total, len(items), pages, size)
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "nessun risultato")
				return nil
			}
			rows := make([]map[string]any, 0, len(items))
			for _, raw := range items {
				var m map[string]any
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				data, _ := m["dataPubblicazione"].(string)
				if len(data) >= 10 {
					data = data[:10]
				}
				rows = append(rows, map[string]any{
					"data":      data,
					"tipologia": m["tipologia"],
					"idAvviso":  m["idAvviso"],
					"cpv":       strings.Join(cpvDeiLotti(m), ", "),
				})
			}
			return printAutoTable(cmd.OutOrStdout(), rows)
		},
	}

	f := cmd.Flags()
	f.StringVar(&cpv, "cpv", "", "Codice CPV o suo prefisso (3-8 cifre), anche con la cifra di controllo (30213000-5); più valori separati da virgola (in OR)")
	f.StringVar(&sa, "sa", "", "Codice fiscale della stazione appaltante")
	f.StringVar(&categorie, "categorie", "", "Categoria lavori come la espone ANAC, spazio compreso (es. \"OG 1\", \"FS\"); i 55 valori ammessi sono in /api/v0/lavori?request=visible. Valorizzata solo sugli avvisi dal 09/07/2026")
	f.StringVar(&from, "from", "", "Data pubblicazione minima GG/MM/AAAA (default 01/01/2024)")
	f.StringVar(&to, "to", "", "Data pubblicazione massima GG/MM/AAAA (default oggi)")
	f.BoolVar(&or, "or", false, "Combina i filtri in OR anziché in AND (ha senso con almeno due filtri fra --cpv, --sa, --categorie)")
	f.IntVar(&size, "size", 20, "Risultati per pagina")
	f.IntVar(&pages, "pages", 1, "Numero di pagine da scaricare")
	return cmd
}

// validateCPVFilter scarta gli input che il server rifiuterebbe, con un errore
// d'uso che nomina il flag. Dal rilascio ANAC del 15/09/2026 il servizio valida
// da sé e risponde HTTP 400 a ogni valore con meno di 3 cifre (anche dentro una
// lista, es. "72,302"), a più di 8 cifre senza trattino (302130005) e a una
// cifra di controllo che non sia una sola cifra (30213000-55). Fino ad agosto
// accettava i prefissi di 2 cifre, cioè le divisioni CPV: non più.
// minCifre vale 3 per i flag che arrivano al servizio (--cpv, --cpv-code) e 2
// per --cpv-exact, che filtra le righe in locale e dove una divisione ha senso.
// La finestra temporale oltre i 12 mesi, che il form del portale blocca, l'API
// la accetta ancora. Il conteggio dichiarato resta inaffidabile per ragioni
// indipendenti dalla lunghezza del prefisso: vedi warnConteggioSottostimato.
func validateCPVFilter(flag, cpv string, minCifre int) error {
	if strings.TrimSpace(cpv) == "" {
		return nil
	}
	for _, tok := range strings.Split(cpv, ",") {
		t := strings.TrimSpace(tok)
		if t == "" {
			continue
		}
		// Il codice come compare negli atti ufficiali porta la cifra di
		// controllo dopo un trattino (30213000-5). Da settembre 2026 il
		// servizio la accetta e la ignora: con la cifra giusta o sbagliata
		// restituisce gli stessi avvisi. Su un prefisso invece non trova
		// nulla e non lo dice, quindi lì il trattino lo rifiutiamo noi.
		base := t
		if i := strings.IndexByte(t, '-'); i >= 0 {
			base = t[:i]
			check := t[i+1:]
			if len(check) != 1 || check[0] < '0' || check[0] > '9' {
				return fmt.Errorf("%s: dopo il trattino serve la sola cifra di controllo (es. 30213000-5), non %q", flag, t)
			}
			if len(base) != 8 {
				return fmt.Errorf("%s: la cifra di controllo vale solo col codice completo a 8 cifre (es. 30213000-5); con un prefisso come %q il servizio non restituisce nulla", flag, t)
			}
		}
		for _, r := range base {
			if r < '0' || r > '9' {
				return fmt.Errorf("%s accetta solo codici numerici (%q non lo è); per cercare una descrizione usa 'cpv search'", flag, t)
			}
		}
		if len(base) < minCifre || len(base) > 8 {
			return fmt.Errorf("%s: ogni valore deve avere da %d a 8 cifre (%q ne ha %d)", flag, minCifre, t, len(base))
		}
	}
	return nil
}

// warnConteggioSottostimato segnala su stderr che il totale dichiarato dal
// server può essere molto inferiore ai risultati realmente ottenibili: `count`
// cresce con la dimensione di pagina richiesta. Verificato il 03/08/2026 sul
// periodo 2024-2026: `cpv=302` dichiara 1.270 risultati con pageSize=1, 1.276
// con 20 (la dimensione che usa il portale) e 12.899 con 200, che è il numero
// effettivo ottenuto scorrendo tutta la paginazione. Sotto il migliaio di
// risultati i valori coincidono, per questo l'avviso scatta solo oltre.
// È il conteggio a essere sbagliato, non l'insieme: i risultati sono pertinenti
// anche in fondo alla paginazione.
// Sui codici completi a 8 cifre il conteggio si è invece sempre rivelato esatto,
// quindi lì l'avviso sarebbe solo rumore.
func warnConteggioSottostimato(w io.Writer, total int64, size int, cpv string) {
	if total < 1000 || size >= 200 || soloCodiciCompleti(cpv) {
		return
	}
	fmt.Fprintf(w, "attenzione: il totale dichiarato da ANAC (%d) dipende da --size e può essere molto inferiore ai risultati realmente ottenibili; ripeti con --size 200 per un conteggio attendibile. L'insieme dei risultati resta corretto\n", total)
}

// warnOrdinamentoIgnorato segnala su stderr che, con testo libero, ANAC ordina
// per rilevanza e ignora sortField/sortDirection: la richiesta li contiene ma
// l'ordine dei risultati non cambia. Verificato il 13/09/2026 con
// "corso intelligenza artificiale": stesso ordine con e senza
// --sort-field dataPubblicazione --sort-dir DESC. Il riordino lato client non
// è un'alternativa, perché ordinerebbe per data solo la pagina già scelta per
// rilevanza; per limitare nel tempo resta il filtro sulla data, che il servizio
// applica.
func warnOrdinamentoIgnorato(w io.Writer, query, sortField, sortDir string) {
	if strings.TrimSpace(query) == "" || (sortField == "" && sortDir == "") {
		return
	}
	fmt.Fprintln(w, "avviso: con --query ANAC ordina per rilevanza e ignora --sort-field/--sort-dir. Per limitare ai più recenti usa --published-from")
}

// soloCodiciCompleti indica se il filtro CPV contiene esclusivamente codici a 8
// cifre, gli unici per cui il conteggio del server è risultato affidabile.
func soloCodiciCompleti(cpv string) bool {
	if strings.TrimSpace(cpv) == "" {
		return false
	}
	for _, t := range codiciCPV(cpv) {
		if len(t) != 8 {
			return false
		}
	}
	return true
}

// codiciCPV divide un filtro CPV sulle virgole e toglie a ogni valore la
// cifra di controllo (30213000-5 -> 30213000): il servizio la ignora, e i
// CPV delle righe normalizzati non la portano.
func codiciCPV(cpv string) []string {
	var out []string
	for _, tok := range strings.Split(cpv, ",") {
		t := strings.TrimSpace(tok)
		if i := strings.IndexByte(t, '-'); i >= 0 {
			t = t[:i]
		}
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// cpvDeiLotti raccoglie i CPV dei lotti di un avviso. L'API li espone in forme
// diverse (descrizione, codice, e nel dettaglio "codice_descrizione"): qui si
// mostra il codice quando si ricava dal vocabolario, altrimenti il valore così
// com'è. Serve a vedere in tabella perché un avviso è stato restituito.
func cpvDeiLotti(item map[string]any) []string {
	var out []string
	seen := map[string]bool{}
	for _, sec := range asArr(templateOf(item)["sections"]) {
		sm := asMap(sec)
		for _, it := range asArr(sm["items"]) {
			raw := asMap(it)["cpv"]
			v := strings.TrimSpace(fmt.Sprint(raw))
			if code, desc, ok := cpvdata.NormalizeCPV(raw); ok && code != "" {
				v = code
			} else if ok && desc != "" {
				v = desc
			}
			if v == "" || v == "<nil>" || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
