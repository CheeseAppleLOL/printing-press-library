// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// cli-printing-press: novel-scaffold-test
// Novel command scaffold tests. Keep the wiring smoke test and add behavior cases as needed.

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/food-and-dining/swiggy/internal/cliutil/testenv"
)

// TestNovelHistoryHelpWires smoke-tests that the history command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelHistoryHelpWires(t *testing.T) {
	testenv.Isolate(t)
	cmd := RootCmd()
	cmd.SetArgs([]string{"history", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("history --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "history", "--since", "Reserved"} {
		if !strings.Contains(help, want) {
			t.Fatalf("history --help missing %q in output:\n%s", want, help)
		}
	}
}

func TestInstamartHistoryCoveragePartial(t *testing.T) {
	full := instamartHistoryCoverage(20, 20)
	if !full.Partial || full.Coverage != "partial_sample" {
		t.Fatalf("20/20 coverage = %+v, want partial_sample", full)
	}
	sample := instamartHistoryCoverage(3, 20)
	if sample.Partial || sample.Coverage != "sample" {
		t.Fatalf("3/20 coverage = %+v, want non-partial sample", sample)
	}
}

func TestBuildHistoryResultSinceReserved(t *testing.T) {
	im := instamartHistoryCoverage(20, 20)
	out := buildHistoryResult(100, 20, map[string]any{"instamart": im.asDomain(100)}, im, "2026-01-01")
	if out["partial"] != true {
		t.Fatalf("partial = %v, want true", out["partial"])
	}
	if out["since_applied"] != false {
		t.Fatalf("since_applied = %v, want false", out["since_applied"])
	}
	if out["since"] != "2026-01-01" {
		t.Fatalf("since = %v", out["since"])
	}
	note, _ := out["since_note"].(string)
	if !strings.Contains(note, "not applied") {
		t.Fatalf("since_note = %q", note)
	}
}
