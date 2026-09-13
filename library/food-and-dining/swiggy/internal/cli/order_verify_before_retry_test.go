// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// cli-printing-press: novel-scaffold-test
// Novel command scaffold tests. Keep the wiring smoke test and add behavior cases as needed.

package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/food-and-dining/swiggy/internal/cliutil/testenv"
)

// TestNovelOrderVerifyBeforeRetryHelpWires smoke-tests that the order verify-before-retry command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelOrderVerifyBeforeRetryHelpWires(t *testing.T) {
	testenv.Isolate(t)
	cmd := RootCmd()
	cmd.SetArgs([]string{"order", "verify-before-retry", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("order verify-before-retry --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "verify-before-retry", "--within"} {
		if !strings.Contains(help, want) {
			t.Fatalf("order verify-before-retry --help missing %q in output:\n%s", want, help)
		}
	}
}

func TestEvaluateRetryMatchTimeWindow(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	window := 30 * time.Minute

	recent := map[string]any{"orderId": "new", "orderTotal": 450.0, "restaurantId": "r_123", "placedAt": now.Add(-5 * time.Minute).Format(time.RFC3339)}
	oldSame := map[string]any{"orderId": "old", "orderTotal": 450.0, "restaurantId": "r_123", "placedAt": now.Add(-48 * time.Hour).Format(time.RFC3339)}
	untimed := map[string]any{"orderId": "undated", "orderTotal": 450.0, "restaurantId": "r_123"}
	otherAmt := map[string]any{"orderId": "other", "orderTotal": 99.0, "placedAt": now.Add(-2 * time.Minute).Format(time.RFC3339)}

	cases := []struct {
		name   string
		orders []map[string]any
		want   retryVerdict
		wantID string
	}{
		{name: "in-window match", orders: []map[string]any{oldSame, recent}, want: retryAlreadyPlaced, wantID: "new"},
		{name: "old same-amount ignored", orders: []map[string]any{oldSame, otherAmt}, want: retrySafe},
		{name: "untimestamped amount match fails closed", orders: []map[string]any{untimed, oldSame}, want: retryAmbiguous, wantID: "undated"},
		{name: "empty history is safe", orders: nil, want: retrySafe},
		{name: "unix timestamp in window", orders: []map[string]any{{"orderId": "unix", "orderTotal": "₹450", "placedAt": float64(now.Add(-2 * time.Minute).Unix())}}, want: retryAlreadyPlaced, wantID: "unix"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateRetryMatch(tc.orders, "450", "", now, window)
			if got.Verdict != tc.want {
				t.Fatalf("verdict = %s (%s), want %s", got.Verdict, got.Reason, tc.want)
			}
			if tc.wantID == "" {
				return
			}
			id, _ := got.Match["orderId"].(string)
			if id != tc.wantID {
				t.Fatalf("matched id = %q, want %q", id, tc.wantID)
			}
		})
	}
}

func TestEvaluateRetryMatchRestaurantFilter(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	older := map[string]any{"orderId": "older", "orderTotal": 450.0, "restaurantId": "r_123", "placedAt": now.Add(-20 * time.Minute).Format(time.RFC3339)}
	newerOther := map[string]any{"orderId": "other-rest", "orderTotal": 450.0, "restaurantId": "r_999", "placedAt": now.Add(-1 * time.Minute).Format(time.RFC3339)}

	got := evaluateRetryMatch([]map[string]any{older, newerOther}, "450", "r_123", now, 30*time.Minute)
	if got.Verdict != retryAlreadyPlaced {
		t.Fatalf("restaurant-filtered in-window match = %s, want already_placed", got.Verdict)
	}
	if got.Match["orderId"] != "older" {
		t.Fatalf("matched %v, want older", got.Match["orderId"])
	}

	wrongRest := evaluateRetryMatch([]map[string]any{newerOther}, "450", "r_123", now, 30*time.Minute)
	if wrongRest.Verdict != retrySafe {
		t.Fatalf("different restaurant should be safe to retry, got %s", wrongRest.Verdict)
	}
}
