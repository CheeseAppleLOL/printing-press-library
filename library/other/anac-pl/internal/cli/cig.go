package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cig"

	"github.com/spf13/cobra"
)

func newCigCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cig",
		Short: "Strumenti sul Codice Identificativo di Gara (CIG)",
		Long:  "Controlli offline sul CIG con l'algoritmo pubblicato da ANAC (anticorruzione/npa, Algoritmo validazione CIG).",
	}
	cmd.AddCommand(newCigCheckCmd(flags))
	return cmd
}

func newCigCheckCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <cig>...",
		Short: "Verifica struttura e cifra di controllo di uno o più CIG, offline",
		Long: strings.Trim(`
Verifica un CIG con l'algoritmo di ANAC, senza chiamare il servizio. Riconosce
le tre famiglie: Simog (iniziale numerica), Simog seconda versione e PCP
(iniziale da A a U) e SmartCIG (iniziale X, Y o Z).

Un CIG che non supera il controllo è stato trascritto male: cercarlo restituisce
avvisi estranei senza dire perché. L'esito di ciascun CIG sta nel campo
valido, con il motivo quando è falso, e il comando esce con 0: così l'esito
arriva intero anche via MCP. In uno script: jq -e 'all(.valido)'. Un argomento
che non ha nemmeno la forma di un CIG (lunghezza diversa da 10, iniziale non
ammessa) è invece un errore d'uso ed esce con 2, senza output.
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli cig check B7E26B1DC7
  anac-pl-pp-cli cig check B7E26B1DC7 Z94375BBCC 5527244A08 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would check CIG codes")
				return nil
			}
			esiti := make([]cig.Esito, 0, len(args))
			for _, a := range args {
				e := cig.Verifica(a)
				if e.Tipo == "" {
					return usageErr(fmt.Errorf("%q non è un CIG: %s", a, e.Motivo))
				}
				esiti = append(esiti, e)
			}
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), esiti, flags)
			}
			rows := make([]map[string]any, 0, len(esiti))
			for _, e := range esiti {
				rows = append(rows, map[string]any{"cig": e.CIG, "valido": e.Valido, "tipo": e.Tipo, "motivo": e.Motivo})
			}
			return printAutoTable(cmd.OutOrStdout(), rows)
		},
	}
	return cmd
}

// warnCIGNonValido segnala su stderr un testo libero che ha la forma di un CIG
// ma non supera la cifra di controllo: la ricerca restituirebbe zero risultati
// o avvisi estranei senza spiegare che il codice è trascritto male. Il testo si
// spezza su tutto ciò che non è lettera o cifra, così un CIG scritto come
// "CIG:B7E26B1DC8" o "(B7E26B1DC8)," viene riconosciuto lo stesso.
func warnCIGNonValido(w io.Writer, query string) {
	separatore := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }
	for _, tok := range strings.FieldsFunc(query, separatore) {
		if !cig.SembraCIG(tok) {
			continue
		}
		if e := cig.Verifica(tok); !e.Valido {
			fmt.Fprintf(w, "avviso: %s ha la forma di un CIG ma non è valido (%s); controlla la trascrizione\n", e.CIG, e.Motivo)
		}
	}
}
