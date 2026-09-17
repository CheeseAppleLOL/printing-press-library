package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/schede"

	"github.com/spf13/cobra"
)

func newTipologieSchedeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schede [codice]...",
		Short: "Descrive i codici scheda degli avvisi (AD3, A1_29, P1_16...), offline",
		Long: strings.Trim(`
Ogni avviso porta un codiceScheda (AD3, A1_29, P1_16, NAG...) che identifica il
modello di pubblicazione e la norma di riferimento. L'elenco è quello pubblicato
da ANAC in anticorruzione/npa (tipologiche/codiceScheda.json, 150 voci).

Senza argomenti elenca tutte le schede; con uno o più codici mostra solo quelli.
Il codice scheda non è il valore da passare a --scheda o --tipologia, che vogliono
il numero template: vedi 'tipologie list'.
`, "\n"),
		Example: strings.Trim(`
  anac-pl-pp-cli tipologie schede AD3
  anac-pl-pp-cli tipologie schede A1_29 P1_16 --json
  anac-pl-pp-cli tipologie schede --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would describe scheda codes")
				return nil
			}
			var out []schede.Scheda
			if len(args) == 0 {
				out = schede.Tutte()
			} else {
				for _, a := range args {
					s, ok := schede.Get(a)
					if !ok {
						return notFoundErr(fmt.Errorf("codice scheda %q non trovato; senza argomenti 'tipologie schede' elenca tutti i codici", a))
					}
					out = append(out, s)
				}
			}
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			rows := make([]map[string]any, 0, len(out))
			for _, s := range out {
				rows = append(rows, map[string]any{"codice": s.Codice, "descrizione": s.Descrizione})
			}
			return printAutoTable(cmd.OutOrStdout(), rows)
		},
	}
	return cmd
}
