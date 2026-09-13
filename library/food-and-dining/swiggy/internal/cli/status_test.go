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

// TestNovelStatusHelpWires smoke-tests that the status command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelStatusHelpWires(t *testing.T) {
	testenv.Isolate(t)
	cmd := RootCmd()
	cmd.SetArgs([]string{"status", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "status"} {
		if !strings.Contains(help, want) {
			t.Fatalf("status --help missing %q in output:\n%s", want, help)
		}
	}
}

func TestLocalTokenStatusUnknownExpiry(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

	missing := localTokenStatus("", time.Time{}, now)
	if missing.Authenticated || !missing.ReauthRecommended || missing.CredentialsPresent {
		t.Fatalf("missing token = %+v", missing)
	}

	unknown := localTokenStatus("tok", time.Time{}, now)
	if unknown.Authenticated || !unknown.CredentialsPresent || unknown.ExpiryKnown || !unknown.ReauthRecommended {
		t.Fatalf("unknown expiry treated as authenticated: %+v", unknown)
	}
	if unknown.ExpiresIn != "unknown" {
		t.Fatalf("expires_in = %q, want unknown", unknown.ExpiresIn)
	}

	expired := localTokenStatus("tok", now.Add(-time.Hour), now)
	if expired.Authenticated || !expired.Expired || !expired.ReauthRecommended {
		t.Fatalf("expired token = %+v", expired)
	}

	ok := localTokenStatus("tok", now.Add(2*time.Hour), now)
	if !ok.Authenticated || ok.ReauthRecommended || ok.ExpiresIn == "" || ok.ExpiryUTC == "" {
		t.Fatalf("valid token = %+v", ok)
	}
}
