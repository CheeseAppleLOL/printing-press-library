// Copyright 2026 Trevin Chow and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/commerce/offerup/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/commerce/offerup/internal/offerup"
)

func newListingsSearchCmd(flags *rootFlags) *cobra.Command {
	lf := &locFlags{}
	var query string
	var limit int
	var postedWithin string
	var priceMin, priceMax float64
	var firmOnly, localOnly bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search live OfferUp listings by keyword and location (no login)",
		Long: strings.Trim(`
Search OfferUp listings near a location. Returns cleaned listings (id, title,
price, location, condition, firm flag, image, URL) with ads filtered out. Set
the area with --zip (or --lat/--lon); narrow with --price-min/--price-max,
--firm, --local, and --category. Results are also cached to the local store so
price-check, deals, and new-since have data.`, "\n"),
		Example:     "  offerup-pp-cli listings search \"dewalt drill\" --zip 98101 --limit 20",
		Annotations: map[string]string{"pp:endpoint": "listings.search", "pp:method": "GET", "pp:path": "/search", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				query = args[0]
			}
			if query == "" && cmd.Flags().NFlag() == 0 && !flags.dryRun {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if query == "" {
				return usageErr(errMissingQuery)
			}
			if cliutil.IsVerifyEnv() {
				return printJSONFiltered(cmd.OutOrStdout(), []offerup.Listing{}, flags)
			}
			opts := lf.searchOpts(limit)
			opts.PriceMin = priceMin
			opts.PriceMax = priceMax
			opts.FirmOnly = firmOnly
			opts.LocalOnly = localOnly
			if postedWithin == "" {
				listings, err := searchAndRecord(cmd, flags, lf, query, opts)
				if err != nil {
					return err
				}
				return printJSONFiltered(cmd.OutOrStdout(), listings, flags)
			}

			_, cutoff, err := postedWithinCutoff(postedWithin)
			if err != nil {
				return usageErr(err)
			}

			searchOpts := opts
			searchOpts.Limit = 0

			client := newOfferupClient(flags)
			candidates, err := client.Search(cmd.Context(), query, searchOpts)
			if err != nil {
				return classifyOfferupError(err)
			}

			listings, err := filterListingsByPostDate(cmd.Context(), client, candidates, cutoff)
			if err != nil {
				return classifyOfferupError(err)
			}

			if limit > 0 && len(listings) > limit {
				listings = listings[:limit]
			}

			if st, err := openOfferupStore(); err == nil {
				defer st.Close()
				_, _ = st.RecordSearch(lf.storeKey(query), listings)
			}

			return printJSONFiltered(cmd.OutOrStdout(), listings, flags)
		},
	}
	cmd.Flags().StringVar(&lf.zip, "zip", "", "ZIP code to scope the search (e.g. 98101)")
	cmd.Flags().StringVar(&lf.lat, "lat", "", "Latitude for precise location (use with --lon)")
	cmd.Flags().StringVar(&lf.lon, "lon", "", "Longitude for precise location (use with --lat)")
	cmd.Flags().StringVar(&lf.city, "city", "", "City name for the search location")
	cmd.Flags().StringVar(&lf.state, "state", "", "Two-letter state code (e.g. WA)")
	cmd.Flags().StringVar(&lf.category, "category", "", "OfferUp category id (cid) to scope the search")
	cmd.Flags().StringVar(&query, "query", "", "Keyword to search for (or pass as a positional argument)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum listings to return (0 returns all on the page)")
	cmd.Flags().StringVar(&postedWithin, "posted-within", "", "Only include listings posted within this duration (for example: 7d, 24h, 1w)")
	cmd.Flags().Float64Var(&priceMin, "price-min", 0, "Only listings at or above this price")
	cmd.Flags().Float64Var(&priceMax, "price-max", 0, "Only listings at or below this price")
	cmd.Flags().BoolVar(&firmOnly, "firm", false, "Only listings with a firm (non-negotiable) price")
	cmd.Flags().BoolVar(&localOnly, "local", false, "Only listings offering local pickup")
	return cmd
}

func postedWithinCutoff(value string) (time.Duration, time.Time, error) {
	d, err := cliutil.ParseDurationLoose(value)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("invalid --posted-within %q: %w", value, err)
	}
	if d <= 0 {
		return 0, time.Time{}, fmt.Errorf("--posted-within must be greater than zero")
	}
	return d, time.Now().Add(-d), nil
}

func filterListingsByPostDate(
	ctx context.Context,
	client *offerup.Client,
	listings []offerup.Listing,
	cutoff time.Time,
) ([]offerup.Listing, error) {
	out := make([]offerup.Listing, 0, len(listings))

	for _, listing := range listings {
		detail, err := client.GetItem(ctx, listing.ListingID)
		if err != nil {
			return nil, fmt.Errorf("fetch listing %s for posting-date filter: %w", listing.ListingID, err)
		}

		if detail == nil || detail.PostDate == "" {
			continue
		}

		postDate, err := time.Parse(time.RFC3339Nano, detail.PostDate)
		if err != nil {
			continue
		}

		if postDate.Before(cutoff) {
			continue
		}

		out = append(out, detail.Listing)
	}

	return out, nil
}
