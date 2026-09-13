// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelOrderCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "order",
		Short:       "Reachability mitigation",
		Example:     "  swiggy-pp-cli order verify-before-retry --domain food --address-id addr_01HXYZ --restaurant-id r_123 --amount 450",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "auto", "pp:happy-args": "verify-before-retry;--domain=food;--address-id=d6tcokq9681fvjepnc60__AbLNWwSYaOosJZH1CJh5-h;--amount=450", "pp:typed-exit-codes": "0,2"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	addNovelCommandIfAbsent(cmd, newNovelOrderVerifyBeforeRetryCmd(flags))
	return cmd
}
