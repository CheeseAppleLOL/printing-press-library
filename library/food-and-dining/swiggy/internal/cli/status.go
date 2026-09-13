// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source computed

package cli

import (
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/food-and-dining/swiggy/internal/config"
	"github.com/spf13/cobra"
)

func newNovelStatusCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "See at a glance whether your Swiggy login is still valid and which domain sessions are active.",
		Long: "Use this to check local credential presence and stored token expiry.\n" +
			"This is not a live Swiggy validation; a revoked token can still look present.\n" +
			"A token without stored expiry (typical for SWIGGY_ACCESS_TOKEN) is treated as unknown and re-auth is recommended.\n" +
			"Do NOT use this to perform login itself; it is read-only. Run 'auth login' to authenticate.",
		Example:     "  swiggy-pp-cli status",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed", "pp:novel-scaffold": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "status")
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}

			report := localTokenStatus(cfg.AccessToken, cfg.TokenExpiry, time.Now())

			w := cmd.OutOrStdout()
			if flags.asJSON {
				out := map[string]any{
					"authenticated":       report.Authenticated,
					"credentials_present": report.CredentialsPresent,
					"token_expiry_known":  report.ExpiryKnown,
					"token_expired":       report.Expired,
					"token_expires_in":    report.ExpiresIn,
					"token_expiry_utc":    report.ExpiryUTC,
					"config_path":         cfg.Path,
					"reauth_recommended":  report.ReauthRecommended,
					"live_check":          false,
				}
				return printJSONFiltered(w, out, flags)
			}

			if !report.CredentialsPresent {
				fmt.Fprintln(w, red("Not authenticated"))
				fmt.Fprintln(w, "  Run 'swiggy-pp-cli auth login' to complete the OAuth 2.1 browser login (phone + OTP).")
				return authErr(fmt.Errorf("%s", report.Error))
			}
			if report.Expired {
				fmt.Fprintln(w, red("Access token expired"))
				fmt.Fprintln(w, "  Swiggy access tokens last 5 days with no refresh-token issuance in v1.0.")
				fmt.Fprintln(w, "  Run 'swiggy-pp-cli auth login' to re-authenticate.")
				return authErr(fmt.Errorf("%s", report.Error))
			}
			if !report.ExpiryKnown {
				fmt.Fprintln(w, red("Credentials present, expiry unknown"))
				fmt.Fprintln(w, "  This token has no stored expiry (typical for SWIGGY_ACCESS_TOKEN).")
				fmt.Fprintln(w, "  Run 'swiggy-pp-cli auth login' to refresh, or treat a mid-flow 401 as re-auth.")
				return authErr(fmt.Errorf("%s", report.Error))
			}
			fmt.Fprintln(w, green("Authenticated"))
			fmt.Fprintf(w, "  Token expires in: %s (%s)\n", report.ExpiresIn, report.ExpiryUTC)
			fmt.Fprintln(w, "  Local expiry only — this command does not call Swiggy to prove the token is still valid.")
			fmt.Fprintln(w, "  Domains share this one session token: food, instamart, dineout each POST to their own endpoint under it.")
			return nil
		},
	}
	return cmd
}

type tokenStatusReport struct {
	Authenticated      bool
	CredentialsPresent bool
	ExpiryKnown        bool
	Expired            bool
	ExpiresIn          string
	ExpiryUTC          string
	ReauthRecommended  bool
	Error              string
}

func localTokenStatus(token string, expiry, now time.Time) tokenStatusReport {
	report := tokenStatusReport{
		CredentialsPresent: token != "",
		ExpiryKnown:        !expiry.IsZero(),
	}
	if !report.CredentialsPresent {
		report.ReauthRecommended = true
		report.Error = "no Swiggy access token stored"
		return report
	}
	if !report.ExpiryKnown {
		report.ExpiresIn = "unknown"
		report.ReauthRecommended = true
		report.Error = "access token present but expiry is unknown; re-auth recommended"
		return report
	}
	report.ExpiryUTC = expiry.UTC().Format(time.RFC3339)
	if !expiry.After(now) {
		report.Expired = true
		report.ExpiresIn = "expired"
		report.ReauthRecommended = true
		report.Error = fmt.Sprintf("access token expired at %s", report.ExpiryUTC)
		return report
	}
	report.Authenticated = true
	report.ExpiresIn = expiry.Sub(now).Round(time.Minute).String()
	return report
}
