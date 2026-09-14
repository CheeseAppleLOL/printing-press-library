package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// newCercaCmd is a hand-authored, human-friendly front end to the
// /avvisi-full-text endpoint. It improves on the raw `avvisi search` by:
//   - accepting a tipologia by name/slug (e.g. "bandi", "esiti") or template id
//   - accepting a free importo range via --amount-min/--amount-max (the web form
//     only offers 4 fixed bands, but the API accepts any min,max)
//   - exposing the search "modalità" (estesa | esatta | archivio) as one flag
//     instead of the two derived params atlasFuzzySearchEnabled / ricercaArchivio
func newCercaCmd(flags *rootFlags) *cobra.Command {
	var query, tipologia, cpv, from, to, mode, sortField, sortDir string
	var amountMin, amountMax string
	var page, size int

	cmd := &cobra.Command{
		Use:   "cerca",
		Short: "Ricerca avvisi con filtri semplici (tipologia per nome, importo min/max, modalità)",
		Long: strings.Trim(`
Ricerca full-text degli avvisi ANAC con filtri facili da usare.

Tipologia: passa un nome o slug (bandi, esiti, indagini-sopra-soglia, ...) oppure
il numero template (4, 7, 5a, ...). Vedi 'tipologie list'.

Importo: --amount-min/--amount-max accettano qualsiasi soglia in euro (non solo
le 4 fasce fisse del sito). Esempi: --amount-min 200000 --amount-max 500000.

Modalità (--mode):
  estesa   (default) ricerca estesa/fuzzy
  esatta   frase esatta: parole adiacenti nell'ordine dato; richiede --tipologia
  archivio cerca nell'archivio storico (usa un intervallo date < 6 mesi)
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli cerca --query microsoft --size 5
  anac-pl-pp-cli cerca --tipologia esiti --query "servizi informatici" --json
  anac-pl-pp-cli cerca --tipologia bandi --amount-min 1000000 --published-from 01/01/2025
  anac-pl-pp-cli cerca --cpv 72000000 --tipologia 7 --agent --select content.idAvviso,content.codiceScheda
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would search avvisi (live)")
				return nil
			}

			params := map[string]string{
				"page": strconv.Itoa(page),
				"size": strconv.Itoa(size),
			}
			if query != "" {
				params["keywords"] = query
			}
			if cpv != "" {
				params["cpv"] = cpv
				// Il campo cpv di /avvisi-full-text è un match testuale sulle
				// descrizioni, non un filtro sul codice del lotto: restituisce
				// anche avvisi di CPV diversi. Vedi docs/note-per-anac.md.
				fmt.Fprintln(cmd.ErrOrStderr(), "avviso: il filtro CPV di 'cerca' non è selettivo (match testuale). Per filtrare davvero per codice usa 'cerca-avanzata --cpv'")
			}
			if from != "" {
				params["dataPubblicazioneStart"] = from
			}
			if to != "" {
				params["dataPubblicazioneEnd"] = to
			}
			if sortField != "" {
				params["sortField"] = sortField
			}
			if sortDir != "" {
				d := strings.ToUpper(sortDir)
				if d != "ASC" && d != "DESC" {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--sort-dir deve essere ASC o DESC"))
				}
				params["sortDirection"] = d
			}
			warnOrdinamentoIgnorato(cmd.ErrOrStderr(), query, sortField, sortDir)

			// tipologia -> codiceScheda (template id)
			if tipologia != "" {
				tpl, ok := resolveTipologia(tipologia)
				if !ok {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("tipologia %q non riconosciuta; vedi 'tipologie list'", tipologia))
				}
				params["codiceScheda"] = tpl
			}

			// importo range -> importoLotto = "min,max" (max 0 = aperto verso l'alto)
			if amountMin != "" || amountMax != "" {
				lo := amountMin
				if lo == "" {
					lo = "0"
				}
				hi := amountMax
				if hi == "" {
					hi = "0"
				}
				if !isAmount(lo) || !isAmount(hi) {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--amount-min/--amount-max devono essere numeri interi (euro)"))
				}
				params["importoLotto"] = lo + "," + hi
			}

			// modalità ricerca
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "", "estesa", "atlas":
				params["atlasFuzzySearchEnabled"] = "true"
			case "esatta", "base":
				params["atlasFuzzySearchEnabled"] = "false"
			case "archivio", "archive":
				params["atlasFuzzySearchEnabled"] = "true"
				params["ricercaArchivio"] = "true"
			default:
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--mode deve essere uno tra: estesa, esatta, archivio"))
			}
			if err := verificaRicercaEsatta(params["atlasFuzzySearchEnabled"] != "false", params["codiceScheda"]); err != nil {
				_ = cmd.Usage()
				return usageErr(err)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := c.Get(cmd.Context(), "/avvisi-full-text", params)
			if err != nil {
				return classifyAPIError(err, flags)
			}

			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				var v any
				if json.Unmarshal(data, &v) != nil {
					return printOutput(cmd.OutOrStdout(), data, true)
				}
				return printJSONFiltered(cmd.OutOrStdout(), v, flags)
			}

			// human: table of the content array
			var env struct {
				Content []map[string]any `json:"content"`
				Count   float64          `json:"count"`
			}
			if json.Unmarshal(data, &env) == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "risultati totali: %d (pagina %d, %d per pagina)\n", int64(env.Count), page, size)
				if len(env.Content) > 0 {
					return printAutoTable(cmd.OutOrStdout(), env.Content)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "nessun risultato")
				return nil
			}
			return printOutput(cmd.OutOrStdout(), data, true)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&query, "query", "q", "", "Testo libero: parola chiave, CIG, CUP, stazione appaltante, oggetto")
	f.StringVarP(&tipologia, "tipologia", "t", "", "Tipologia avviso: nome/slug (bandi, esiti, ...) o template (4, 7, 5a). Vedi 'tipologie list'")
	f.StringVar(&cpv, "cpv", "", "Codice CPV (vedi 'cpv search')")
	f.StringVar(&amountMin, "amount-min", "", "Importo lotto minimo in euro (range libero)")
	f.StringVar(&amountMax, "amount-max", "", "Importo lotto massimo in euro (range libero; vuoto/0 = nessun limite)")
	f.StringVar(&from, "published-from", "", "Data pubblicazione minima, formato GG/MM/AAAA")
	f.StringVar(&to, "published-to", "", "Data pubblicazione massima, formato GG/MM/AAAA")
	f.StringVar(&mode, "mode", "estesa", "Modalità ricerca: estesa | esatta | archivio")
	f.StringVar(&sortField, "sort-field", "", "Campo di ordinamento (es. dataPubblicazione). Il servizio lo onora solo senza --query: con testo libero ordina per rilevanza")
	f.StringVar(&sortDir, "sort-dir", "", "Direzione ordinamento: ASC o DESC")
	f.IntVar(&page, "page", 0, "Numero pagina (0-based)")
	f.IntVar(&size, "size", 10, "Risultati per pagina")
	return cmd
}

func isAmount(s string) bool {
	if s == "" {
		return false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err != nil {
		return false
	}
	return true
}
