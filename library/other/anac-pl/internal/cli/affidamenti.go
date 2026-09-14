package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cpvdata"
	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/giurisdizione"

	"github.com/spf13/cobra"
)

// affidamentoRow è una riga appiattita: un lotto per un aggiudicatario.
type affidamentoRow struct {
	Data             string  `json:"data"`
	Committente      string  `json:"committente"`
	CFCommittente    string  `json:"cf_committente"`
	Aggiudicatario   string  `json:"aggiudicatario"`
	CFAggiudicatario string  `json:"cf_aggiudicatario"`
	Giurisdizione    string  `json:"giurisdizione"`
	Gruppo           string  `json:"gruppo"`
	Importo          float64 `json:"importo"`
	// ImportoNoto distingue un importo pubblicato, anche zero, da uno assente
	// o illeggibile, che in Importo vale 0: serve all'ordinamento.
	ImportoNoto bool   `json:"-"`
	CIG         string `json:"cig"`
	CPV         string `json:"cpv"`
	CPVDesc     string `json:"cpv_desc"`
	IDAvviso    string `json:"id_avviso"`
	IDAppalto   string `json:"id_appalto"`
}

// newAffidamentiCmd appiattisce gli esiti di gara in una tabella analizzabile
// (committente -> aggiudicatario -> importo) con il CPV normalizzato e la
// giurisdizione del fornitore. Pensato per analisi di sovranità digitale:
// quanto spende la PA, e verso fornitori di quale giurisdizione.
func newAffidamentiCmd(flags *rootFlags) *cobra.Command {
	var query, tipologia, cpv, cpvExact, cpvCode, from, to, mode, jur string
	var amountMin, amountMax string
	var sortField, sortDir string
	var size, pages int
	var fromSearch bool

	cmd := &cobra.Command{
		Use:   "affidamenti",
		Short: "Tabella appiattita degli affidamenti: committente, aggiudicatario, importo, CPV, giurisdizione",
		Long: strings.Trim(`
Estrae dagli esiti di gara una riga per ogni lotto/aggiudicatario, con:
committente, codice fiscale, aggiudicatario, importo, CIG, CPV (normalizzato dal
vocabolario UE) e giurisdizione del fornitore (IT / UE / EXTRA-UE).

La giurisdizione si basa sulla nazionalità del fornitore (gruppo di
appartenenza), non sulla collocazione del dato. Utile per misurare la dipendenza
della PA da provider extra-UE (es. email/cloud).

Con --cpv-code la ricerca passa dall'endpoint della "Ricerca avanzata" ANAC, il
cui filtro CPV seleziona davvero per codice (o prefisso) su tutto l'archivio.
Quell'endpoint non conosce testo libero, importi né tipologia: --query e
--amount-* vengono rifiutati, mentre --tipologia (default 'esiti') è applicato
lato client sui risultati scaricati, quindi conviene alzare --pages. Per non
filtrare per tipologia passa -t "".

Ordinamento (--sort-field/--sort-dir): le chiavi sono le colonne della tabella
(data, importo, committente, aggiudicatario, cpv, giurisdizione, cig, id_avviso)
e l'ordine è applicato alle righe estratte. Per 'data' la richiesta arriva anche
al servizio, così le pagine scaricate sono davvero le prime in quell'ordine: vale
con --cpv-code e, sulla vecchia ricerca, solo senza --query. Con --query il
servizio ordina per rilevanza e ignora la richiesta: l'ordine copre allora solo
le pagine scaricate (--pages x --size), non l'intero archivio.

Doppi invii: la piattaforma pubblica anche gli avvisi mandati due volte dalla
stazione appaltante (stesso id_appalto, stessa scheda, contenuto identico, a
pochi secondi di distanza), e qui compaiono come righe uguali con id_avviso
diverso. Non vengono fusi, perché sullo stesso CIG possono esistere avvisi
diversi e legittimi (esito, rettifica, ripubblicazione): per riconoscerli
usa id_appalto insieme a cig, cf_aggiudicatario, importo e data.

Caveat: ANAC non copre i tier gratuiti (es. Workspace for Education) né gli
affidamenti aggregati via Consip; incrociare con altre fonti (es. MxMap).
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli affidamenti -q "posta elettronica" -t esiti --csv > affidamenti.csv
  anac-pl-pp-cli affidamenti --cpv-code 72412000 -t "" --pages 3 --from-search --csv
  anac-pl-pp-cli affidamenti -q microsoft -t esiti --jurisdiction extra-ue
  anac-pl-pp-cli affidamenti -q "google workspace" --pages 3 --json
  anac-pl-pp-cli affidamenti -q "google workspace" --pages 3 --sort-field data --sort-dir desc --csv
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would fetch and flatten affidamenti (live)")
				return nil
			}
			sortKey, sortDesc, err := parseAffidamentiSort(sortField, sortDir)
			if err != nil {
				_ = cmd.Usage()
				return usageErr(err)
			}
			// --cpv-code usa la ricerca avanzata, che accetta solo date, CPV e
			// stazione appaltante: gli altri filtri non hanno un equivalente.
			wantTipologia := ""
			if cpvCode != "" {
				if err := validateCPVFilter(cpvCode); err != nil {
					_ = cmd.Usage()
					return usageErr(err)
				}
				if query != "" {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--cpv-code e --query non sono combinabili: la ricerca per codice CPV non supporta il testo libero"))
				}
				if amountMin != "" || amountMax != "" {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--cpv-code non è combinabile con --amount-min/--amount-max: filtra gli importi a valle (es. con qsv o duckdb sul CSV)"))
				}
				if cpv != "" {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("usa --cpv-code oppure --cpv, non entrambi"))
				}
				if cmd.Flags().Changed("mode") {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--mode non si applica a --cpv-code: la ricerca avanzata non ha modalità estesa/esatta/archivio"))
				}
				if tipologia != "" {
					tipo, ok := resolveTipologiaTipo(tipologia)
					if !ok {
						_ = cmd.Usage()
						return usageErr(fmt.Errorf("tipologia %q non riconosciuta; vedi 'tipologie list'", tipologia))
					}
					wantTipologia = tipo
				}
			}

			base := map[string]string{}
			if query != "" {
				base["keywords"] = query
			}
			if from != "" {
				base["dataPubblicazioneStart"] = from
			}
			if to != "" {
				base["dataPubblicazioneEnd"] = to
			}
			// La vecchia ricerca onora sortField/sortDirection solo senza
			// keywords (misurato il 04/09/2026: con testo libero asc e desc
			// restituiscono le stesse pagine, per rilevanza). Li si manda
			// quindi solo quando servono a qualcosa.
			if sortKey == "data" && query == "" {
				base["sortField"] = "dataPubblicazione"
				base["sortDirection"] = sortDirParam(sortDesc)
			}
			if tipologia != "" {
				tpl, ok := resolveTipologia(tipologia)
				if !ok {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("tipologia %q non riconosciuta; vedi 'tipologie list'", tipologia))
				}
				base["codiceScheda"] = tpl
			}
			if amountMin != "" || amountMax != "" {
				lo, hi := amountMin, amountMax
				if lo == "" {
					lo = "0"
				}
				if hi == "" {
					hi = "0"
				}
				if !isAmount(lo) || !isAmount(hi) {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--amount-min/--amount-max devono essere interi (euro)"))
				}
				base["importoLotto"] = lo + "," + hi
			}
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "", "estesa", "atlas":
				base["atlasFuzzySearchEnabled"] = "true"
			case "esatta", "base":
				base["atlasFuzzySearchEnabled"] = "false"
			case "archivio", "archive":
				base["atlasFuzzySearchEnabled"] = "true"
				base["ricercaArchivio"] = "true"
			default:
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--mode deve essere: estesa, esatta, archivio"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			if size <= 0 {
				size = 20
			}
			if pages <= 0 {
				pages = 1
			}
			jurFilter := strings.ToUpper(strings.TrimSpace(jur))

			// 1) raccogli gli avvisi. Con --cpv-code si usa la ricerca avanzata
			// (/avvisi-full-text-specializzata), il cui filtro CPV è selettivo sul
			// codice del lotto: niente più doppia query codice+descrizione né
			// dipendenza dal testo. Altrimenti si resta sulla vecchia full-text.
			var ids []string
			var rawItems []json.RawMessage
			seen := map[string]bool{}

			cpvExactAuto := false
			if cpvCode != "" {
				if cpvExact == "" {
					cpvExact = cpvCode // tiene solo i lotti di quel CPV
					cpvExactAuto = true
				}
				sp := specializzataParams(cpvCode, "", "", from, to, false, size)
				if sortKey == "data" {
					sp["sortDirection"] = sortDirParam(sortDesc)
				}
				items, _, err := fetchSpecializzata(cmd.Context(), c, sp, pages)
				if err != nil {
					return classifyAPIError(err, flags)
				}
				for _, raw := range items {
					var head struct {
						IDAvviso  string `json:"idAvviso"`
						Tipologia string `json:"tipologia"`
					}
					_ = json.Unmarshal(raw, &head)
					if head.IDAvviso == "" || seen[head.IDAvviso] {
						continue
					}
					// la ricerca avanzata non filtra per tipologia: lo facciamo qui
					if wantTipologia != "" && !strings.EqualFold(head.Tipologia, wantTipologia) {
						continue
					}
					seen[head.IDAvviso] = true
					ids = append(ids, head.IDAvviso)
					rawItems = append(rawItems, raw)
				}
			} else {
				// L'API usa paginazione a TOKEN, non a numero di pagina: si passa
				// direzionePaginazione=AVANTI + tokenPaginazione=<lastPaginationToken>.
				token := ""
				for p := 0; p < pages; p++ {
					params := map[string]string{"size": strconv.Itoa(size)}
					for k, v := range base {
						params[k] = v
					}
					if cpv != "" {
						params["cpv"] = cpv
					}
					if token != "" {
						params["direzionePaginazione"] = "AVANTI"
						params["tokenPaginazione"] = token
					}
					data, err := c.Get(cmd.Context(), "/avvisi-full-text", params)
					if err != nil {
						return classifyAPIError(err, flags)
					}
					var env struct {
						Content             []json.RawMessage `json:"content"`
						LastPaginationToken string            `json:"lastPaginationToken"`
					}
					if json.Unmarshal(data, &env) != nil || len(env.Content) == 0 {
						break
					}
					for _, raw := range env.Content {
						var idOnly struct {
							IDAvviso string `json:"idAvviso"`
						}
						_ = json.Unmarshal(raw, &idOnly)
						if idOnly.IDAvviso == "" || seen[idOnly.IDAvviso] {
							continue
						}
						seen[idOnly.IDAvviso] = true
						ids = append(ids, idOnly.IDAvviso)
						rawItems = append(rawItems, raw)
					}
					if len(env.Content) < size || env.LastPaginationToken == "" {
						break
					}
					token = env.LastPaginationToken
				}
			}

			// 2) appiattisci. Con --from-search si usa direttamente il contenuto
			//    della ricerca (che già embedda committente, aggiudicatari, CPV e
			//    importo): nessuna chiamata di dettaglio, molto più veloce. Senza,
			//    si scarica il dettaglio /avvisi/{id} di ogni avviso (più completo
			//    per i record eForms con CPV in forma di codice).
			var rows []affidamentoRow
			if fromSearch {
				for _, raw := range rawItems {
					var item map[string]any
					if json.Unmarshal(raw, &item) != nil {
						continue
					}
					rows = append(rows, flattenAvviso(item)...)
				}
			} else {
				for _, id := range ids {
					d, err := c.Get(cmd.Context(), "/avvisi/"+id, nil)
					if err != nil {
						continue
					}
					var item map[string]any
					if json.Unmarshal(d, &item) != nil {
						continue
					}
					rows = append(rows, flattenAvviso(item)...)
				}
			}

			// filtro CPV LATO CLIENT sul codice reale del lotto. Con --cpv-code
			// serve solo a scartare gli altri lotti di un avviso multi-lotto (la
			// selezione l'ha già fatta il server): in quel caso teniamo anche le
			// righe il cui CPV non è normalizzabile, altrimenti perderemmo lotti
			// che l'API ha correttamente restituito. Se invece --cpv-exact è
			// esplicito, l'utente vuole un filtro stretto.
			if cpvExact != "" {
				want := strings.TrimSpace(cpvExact)
				kept := rows[:0]
				for _, r := range rows {
					if r.CPV == want || strings.HasPrefix(r.CPV, want) || (cpvExactAuto && r.CPV == "") {
						kept = append(kept, r)
					}
				}
				rows = kept
			}

			// filtro giurisdizione opzionale
			if jurFilter != "" {
				kept := rows[:0]
				for _, r := range rows {
					if strings.ToUpper(r.Giurisdizione) == jurFilter {
						kept = append(kept, r)
					}
				}
				rows = kept
			}

			// Ultimo passo: un avviso genera più righe (lotto x aggiudicatario)
			// e il servizio sa ordinare solo gli avvisi.
			sortAffidamenti(rows, sortKey, sortDesc)

			return emitAffidamenti(cmd, flags, rows)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&query, "query", "q", "", "Testo libero (keyword, CIG, CUP, oggetto)")
	f.StringVarP(&tipologia, "tipologia", "t", "esiti", "Tipologia (nome/slug o template); default 'esiti'")
	f.StringVar(&cpv, "cpv", "", "Valore grezzo per il campo CPV della vecchia ricerca (match testuale, non selettivo; di norma usa --cpv-code)")
	f.StringVar(&cpvCode, "cpv-code", "", "Codice CPV o suo prefisso (min 3 cifre): usa la ricerca avanzata ANAC, che filtra davvero per codice. Non combinabile con --query/--amount-*")
	f.StringVar(&cpvExact, "cpv-exact", "", "Filtro CPV ESATTO lato client sul codice reale (es. 72212220; accetta prefisso, es. 72)")
	f.StringVar(&amountMin, "amount-min", "", "Importo minimo (euro)")
	f.StringVar(&amountMax, "amount-max", "", "Importo massimo (euro)")
	f.StringVar(&from, "published-from", "", "Data pubblicazione minima GG/MM/AAAA")
	f.StringVar(&to, "published-to", "", "Data pubblicazione massima GG/MM/AAAA")
	f.StringVar(&mode, "mode", "estesa", "Modalità: estesa | esatta | archivio")
	f.StringVar(&jur, "jurisdiction", "", "Filtra per giurisdizione fornitore: IT | UE | EXTRA-UE")
	f.IntVar(&size, "size", 20, "Risultati per pagina")
	f.IntVar(&pages, "pages", 1, "Numero di pagine da scaricare")
	f.BoolVar(&fromSearch, "from-search", false, "Appiattisci dai risultati di ricerca senza scaricare il dettaglio per record (molto più veloce)")
	f.StringVar(&sortField, "sort-field", "", "Colonna di ordinamento: "+strings.Join(affidamentiSortFields, ", "))
	f.StringVar(&sortDir, "sort-dir", "", "Direzione ordinamento: ASC (default) o DESC")
	return cmd
}

// affidamentiSortFields sono le colonne su cui si può ordinare: gli stessi
// nomi dell'intestazione CSV e delle chiavi JSON.
var affidamentiSortFields = []string{"data", "importo", "committente", "aggiudicatario", "cpv", "giurisdizione", "cig", "id_avviso", "id_appalto"}

// parseAffidamentiSort valida --sort-field/--sort-dir. Campo vuoto = nessun
// ordinamento; la direzione da sola non basta.
func parseAffidamentiSort(field, dir string) (string, bool, error) {
	field = strings.ToLower(strings.TrimSpace(field))
	dir = strings.ToUpper(strings.TrimSpace(dir))
	if field == "" {
		if dir != "" {
			return "", false, fmt.Errorf("--sort-dir richiede --sort-field")
		}
		return "", false, nil
	}
	ok := false
	for _, f := range affidamentiSortFields {
		if f == field {
			ok = true
			break
		}
	}
	if !ok {
		return "", false, fmt.Errorf("--sort-field %q non valido; usa uno di: %s", field, strings.Join(affidamentiSortFields, ", "))
	}
	switch dir {
	case "", "ASC":
		return field, false, nil
	case "DESC":
		return field, true, nil
	}
	return "", false, fmt.Errorf("--sort-dir deve essere ASC o DESC")
}

func sortDirParam(desc bool) string {
	if desc {
		return "desc"
	}
	return "asc"
}

// sortAffidamenti ordina le righe sul campo scelto. Stabile, così a parità di
// chiave resta l'ordine del servizio. I valori vuoti vanno in fondo in
// entrambe le direzioni: una data mancante non è "la più vecchia".
func sortAffidamenti(rows []affidamentoRow, field string, desc bool) {
	if field == "" {
		return
	}
	key := func(r affidamentoRow) string {
		switch field {
		case "data":
			return r.Data
		case "committente":
			return r.Committente
		case "aggiudicatario":
			return r.Aggiudicatario
		case "cpv":
			return r.CPV
		case "giurisdizione":
			return r.Giurisdizione
		case "cig":
			return r.CIG
		case "id_avviso":
			return r.IDAvviso
		case "id_appalto":
			return r.IDAppalto
		}
		return ""
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if field == "importo" {
			// Senza importo in fondo in entrambe le direzioni, come i valori
			// vuoti degli altri campi: altrimenti in ordine crescente il loro
			// 0 li metterebbe prima di ogni aggiudicazione vera.
			if rows[i].ImportoNoto != rows[j].ImportoNoto {
				return rows[i].ImportoNoto
			}
			if desc {
				return rows[i].Importo > rows[j].Importo
			}
			return rows[i].Importo < rows[j].Importo
		}
		a, b := key(rows[i]), key(rows[j])
		if (a == "") != (b == "") {
			return a != ""
		}
		if desc {
			return a > b
		}
		return a < b
	})
}

func emitAffidamenti(cmd *cobra.Command, flags *rootFlags, rows []affidamentoRow) error {
	if rows == nil {
		rows = []affidamentoRow{}
	}
	if flags.csv {
		w := csv.NewWriter(cmd.OutOrStdout())
		_ = w.Write([]string{"data", "committente", "cf_committente", "aggiudicatario", "cf_aggiudicatario", "giurisdizione", "gruppo", "importo", "cig", "cpv", "cpv_desc", "id_avviso", "id_appalto"})
		for _, r := range rows {
			_ = w.Write([]string{r.Data, r.Committente, r.CFCommittente, r.Aggiudicatario, r.CFAggiudicatario, r.Giurisdizione, r.Gruppo, formatImporto(r.Importo), r.CIG, r.CPV, r.CPVDesc, r.IDAvviso, r.IDAppalto})
		}
		w.Flush()
		return w.Error()
	}
	if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.quiet && !flags.plain) {
		return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
	}
	if len(rows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "nessun affidamento estratto")
		return nil
	}
	items := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]any{
			"committente":    trunc(r.Committente, 40),
			"aggiudicatario": trunc(r.Aggiudicatario, 30),
			"giurisd":        r.Giurisdizione,
			"importo":        formatImporto(r.Importo),
			"cpv":            r.CPV,
			"cig":            r.CIG,
		})
	}
	return printAutoTable(cmd.OutOrStdout(), items)
}

// flattenAvviso estrae le righe (lotto × aggiudicatario) da un avviso.
func flattenAvviso(item map[string]any) []affidamentoRow {
	id, _ := item["idAvviso"].(string)
	appalto, _ := item["idAppalto"].(string)
	data, _ := item["dataPubblicazione"].(string)
	if len(data) >= 10 {
		data = data[:10]
	}
	tmpl := templateOf(item)
	comm, cfComm := committenteFrom(tmpl)
	comm, cfComm = cleanStr(comm), cleanStr(cfComm)

	var out []affidamentoRow
	for _, sec := range asArr(tmpl["sections"]) {
		sm := asMap(sec)
		if !strings.Contains(fmt.Sprint(sm["name"]), "SEZ. C") {
			continue
		}
		for _, it := range asArr(sm["items"]) {
			im := asMap(it)
			cig := str(im["cig"])
			code, desc, _ := cpvdata.NormalizeCPV(im["cpv"])
			vendors := vendorsFrom(im)
			if len(vendors) == 0 {
				imp, noto := importoFrom(im, nil)
				out = append(out, affidamentoRow{
					Data: data, Committente: comm, CFCommittente: cfComm,
					Giurisdizione: giurisdizione.Ignota, Importo: imp, ImportoNoto: noto,
					CIG: cig, CPV: code, CPVDesc: desc, IDAvviso: id, IDAppalto: appalto,
				})
				continue
			}
			for _, v := range vendors {
				name := cleanStr(v.name)
				g := giurisdizione.Classify(name)
				imp, noto := importoFrom(im, v.importo)
				out = append(out, affidamentoRow{
					Data: data, Committente: comm, CFCommittente: cfComm,
					Aggiudicatario: name, CFAggiudicatario: cleanStr(v.cf),
					Giurisdizione: g.Giurisdizione, Gruppo: g.Gruppo,
					Importo: imp, ImportoNoto: noto,
					CIG: cig, CPV: code, CPVDesc: desc, IDAvviso: id, IDAppalto: appalto,
				})
			}
		}
	}
	return out
}

type vendorInfo struct {
	name    string
	cf      string
	importo *float64
}

func vendorsFrom(item map[string]any) []vendorInfo {
	var out []vendorInfo
	// schema eForms usa "aggiudicatari"; schema vecchio usa "aggiudicatari_ad"
	ad := item["aggiudicatari"]
	if ad == nil {
		ad = item["aggiudicatari_ad"]
	}
	var groups []any
	switch x := ad.(type) {
	case []any:
		groups = x
	case map[string]any:
		groups = []any{x}
	default:
		return out
	}
	for _, g := range groups {
		gm := asMap(g)
		var imp *float64
		if f, ok := toFloat(gm["importo"]); ok {
			imp = &f
		}
		sogg := asArr(gm["soggetti"])
		if len(sogg) == 0 {
			if name := getStr(gm, "denominazione", "denominazione_amministrazione", "ragione_sociale"); name != "" {
				out = append(out, vendorInfo{name: name, cf: getStr(gm, "codice_fiscale", "cf"), importo: imp})
			}
			continue
		}
		for _, s := range sogg {
			smap := asMap(s)
			name := getStr(smap, "denominazione", "denominazione_amministrazione", "ragione_sociale")
			if name == "" {
				continue
			}
			out = append(out, vendorInfo{name: name, cf: getStr(smap, "codice_fiscale", "cf"), importo: imp})
		}
	}
	return out
}

func committenteFrom(tmpl map[string]any) (string, string) {
	for _, sec := range asArr(tmpl["sections"]) {
		sm := asMap(sec)
		if !strings.Contains(fmt.Sprint(sm["name"]), "SEZ. A") {
			continue
		}
		f := asMap(sm["fields"])
		if sa := asArr(f["soggetti_sa"]); len(sa) > 0 {
			m := asMap(sa[0])
			return getStr(m, "denominazione_amministrazione", "denominazione"), getStr(m, "codice_fiscale", "cf")
		}
		if d := getStr(f, "denominazione_sa", "denominazione_amministrazione"); d != "" {
			return d, getStr(f, "codice_fiscale_sa", "codice_fiscale")
		}
		// soggetti_sa come oggetto singolo
		if m := asMap(f["soggetti_sa"]); len(m) > 0 {
			return getStr(m, "denominazione_amministrazione", "denominazione"), getStr(m, "codice_fiscale", "cf")
		}
	}
	return "", ""
}

// templateOf estrae il primo template di un avviso. Il vecchio endpoint
// /avvisi-full-text lo espone sotto "template", la ricerca avanzata sotto
// "templates": accettiamo entrambe le chiavi.
func templateOf(item map[string]any) map[string]any {
	for _, key := range []string{"template", "templates"} {
		for _, t := range asArr(item[key]) {
			tm := asMap(t)
			if inner := asMap(tm["template"]); len(inner) > 0 {
				return inner
			}
		}
	}
	return map[string]any{}
}

func importoFrom(item map[string]any, vendorImp *float64) (float64, bool) {
	if vendorImp != nil {
		return *vendorImp, true
	}
	// prova i vari nomi campo dei due schemi, in ordine di preferenza
	for _, k := range []string{"valore_offerta_vincente", "valore_affidamento", "valore_complessivo_stimato"} {
		switch x := item[k].(type) {
		case map[string]any:
			if f, ok := toFloat(x["value"]); ok {
				return f, true
			}
			if f, ok := toFloat(x["importo"]); ok {
				return f, true
			}
		case nil:
			// campo assente, prova il prossimo
		default:
			if f, ok := toFloat(x); ok {
				return f, true
			}
		}
	}
	return 0, false
}

// --- helper generici per JSON eterogeneo ---

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asArr(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return nil
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func formatImporto(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// cleanStr normalizza una stringa estratta dal JSON: rimuove a-capo e spazi
// multipli, così il CSV non contiene celle con newline incorporati.
func cleanStr(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
