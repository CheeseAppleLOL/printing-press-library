package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cpvdata"

	"github.com/spf13/cobra"
)

// newCpvCmd is a hand-authored offline reference for the EU Common Procurement
// Vocabulary (CPV 2008) with Italian descriptions, embedded in the binary.
// It lets users find the right CPV code to pass to `avvisi search --cpv` /
// `cerca --cpv` without an internet round-trip.
func newCpvCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cpv",
		Short: "Sfoglia e cerca i codici CPV (vocabolario appalti UE) con descrizione italiana",
		Long: "Riferimento offline del Common Procurement Vocabulary (CPV 2008, Reg. CE 213/2008): " +
			fmt.Sprintf("%d codici con descrizione ufficiale italiana, embeddati nel binario.\n", cpvdata.Count()) +
			"Usa 'cpv search <testo|codice>' per trovare un codice, poi passalo a 'avvisi search --cpv' o 'cerca --cpv'.",
		Annotations: map[string]string{"mcp:read-only": "true"},
	}
	cmd.AddCommand(newCpvSearchCmd(flags))
	cmd.AddCommand(newCpvGetCmd(flags))
	return cmd
}

func newCpvSearchCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "search [testo o codice...]",
		Short: "Cerca codici CPV per descrizione o per prefisso di codice",
		Long: "Cerca nel vocabolario CPV. Una query numerica filtra per prefisso di codice " +
			"(es. '72' -> tutti i servizi IT); una query testuale richiede che ogni parola " +
			"compaia nella descrizione (es. 'licenze software').",
		Example: strings.Trim(`
  anac-pl-pp-cli cpv search software
  anac-pl-pp-cli cpv search licenze software --limit 30
  anac-pl-pp-cli cpv search 72 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would search CPV vocabulary")
				return nil
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			if query == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("specifica un testo o un codice da cercare"))
			}
			results := cpvdata.Search(query, limit)
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), results, flags)
			}
			if len(results) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "nessun codice CPV trovato per %q\n", query)
				return nil
			}
			rows := make([]map[string]any, 0, len(results))
			for _, e := range results {
				rows = append(rows, map[string]any{"code": e.Code, "description": e.Description, "level": e.Level})
			}
			return printAutoTable(cmd.OutOrStdout(), rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "numero massimo di risultati (0 = tutti)")
	return cmd
}

func newCpvGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "get <codice>",
		Short:       "Mostra la descrizione di un codice CPV esatto",
		Example:     "  anac-pl-pp-cli cpv get 72000000\n  anac-pl-pp-cli cpv get 72000000-5 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would look up a CPV code")
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("specifica un codice CPV (es. 72000000)"))
			}
			e, ok := cpvdata.Get(args[0])
			if !ok {
				return notFoundErr(fmt.Errorf("codice CPV %q non trovato", args[0]))
			}
			if flags.asJSON || flags.agent || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), e, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", e.Code, e.Description)
			return nil
		},
	}
	return cmd
}
