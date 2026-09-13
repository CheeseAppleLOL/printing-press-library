// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelPayCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "pay",
		Short:       "Agent-native plumbing",
		Example:     "  swiggy-pp-cli pay wait --paas-id paas_123 --order-id ord_01HXYZ --domain food --max-wait 5s",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "auto", "pp:typed-exit-codes": "0,1,2"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	addNovelCommandIfAbsent(cmd, newNovelPayWaitCmd(flags))
	return cmd
}
