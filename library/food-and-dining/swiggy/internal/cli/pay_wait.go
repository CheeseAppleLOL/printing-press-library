// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source live

package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newNovelPayWaitCmd(flags *rootFlags) *cobra.Command {
	var flagPaasId string
	var flagOrderId string
	var flagAddressId string
	var flagLat float64
	var flagLng float64
	var flagDomain string
	var flagMaxWait time.Duration

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Block until a UPI payment resolves instead of hand-rolling a polling loop yourself.",
		Long: "Use this after placing an order paid via UPI to block until payment resolves.\n" +
			"Do NOT use this for COD orders; call the domain's check-payment-status once and proceed.",
		Example:     "  swiggy-pp-cli pay wait --paas-id paas_123 --order-id ord_01HXYZ --domain food --max-wait 5s",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "live", "pp:novel-scaffold": "true", "pp:happy-args": "--paas-id=paas_123;--order-id=ord_01HXYZ;--domain=food;--max-wait=5s", "pp:typed-exit-codes": "0,1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "pay wait")
			}
			if flagPaasId == "" {
				return fmt.Errorf("required flag \"paas-id\" not set")
			}
			path, ok := swiggyDomainPath(flagDomain)
			if !ok {
				return fmt.Errorf("invalid --domain %q; must be one of food, instamart, dineout", flagDomain)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			toolArgs := map[string]any{"paasId": flagPaasId}
			if flagOrderId != "" {
				toolArgs["orderId"] = flagOrderId
			}
			if flagAddressId != "" {
				toolArgs["addressId"] = flagAddressId
			}
			if flagLat != 0 {
				toolArgs["lat"] = flagLat
			}
			if flagLng != 0 {
				toolArgs["lng"] = flagLng
			}

			deadline := time.Now().Add(flagMaxWait)
			var last json.RawMessage
			for {
				data, _, err := c.MCPToolQuery(cmd.Context(), path, "check_payment_status", nil, toolArgs)
				if err != nil {
					return classifyAPIError(cmd.OutOrStdout(), err, flags)
				}
				last = data
				terminal, _, err := parsePaymentTerminal(data)
				if err != nil {
					return err
				}
				if terminal {
					break
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("pay wait: payment did not reach a terminal state within %s", flagMaxWait)
				}
				// The server's own long-poll holds ~19s per the docs; this guard
				// only prevents a tight loop if a given deployment responds fast.
				select {
				case <-time.After(1 * time.Second):
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				}
			}
			return printOutputWithFlagsMeta(cmd.OutOrStdout(), last, flags, map[string]any{"source": "live"}, nil)
		},
	}
	cmd.Flags().StringVar(&flagPaasId, "paas-id", "", "Payment session id (paasId) returned by the place-order/checkout/book-table call")
	cmd.Flags().StringVar(&flagOrderId, "order-id", "", "Order id returned alongside paasId")
	cmd.Flags().StringVar(&flagAddressId, "address-id", "", "Required for Food: the same addressId used for the order")
	cmd.Flags().Float64Var(&flagLat, "lat", 0, "Required for Food reconciliation alongside addressId")
	cmd.Flags().Float64Var(&flagLng, "lng", 0, "Required for Food reconciliation alongside addressId")
	cmd.Flags().StringVar(&flagDomain, "domain", "", "One of food, instamart, dineout")
	cmd.Flags().DurationVar(&flagMaxWait, "max-wait", 2*time.Minute, "Give up after this long if payment never reaches a terminal state")
	return cmd
}

func parsePaymentTerminal(data []byte) (bool, string, error) {
	var parsed struct {
		Data *struct {
			Terminal *bool  `json:"terminal"`
			Status   string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false, "", fmt.Errorf("pay wait: payment status response was not valid JSON: %w", err)
	}
	if parsed.Data == nil || parsed.Data.Terminal == nil {
		return false, "", fmt.Errorf("pay wait: payment status response missing data.terminal")
	}
	return *parsed.Data.Terminal, parsed.Data.Status, nil
}
