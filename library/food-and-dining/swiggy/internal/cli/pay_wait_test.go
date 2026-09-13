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

// TestNovelPayWaitHelpWires smoke-tests that the pay wait command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelPayWaitHelpWires(t *testing.T) {
	testenv.Isolate(t)
	cmd := RootCmd()
	cmd.SetArgs([]string{"pay", "wait", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pay wait --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "wait"} {
		if !strings.Contains(help, want) {
			t.Fatalf("pay wait --help missing %q in output:\n%s", want, help)
		}
	}
}

func TestParsePaymentTerminal(t *testing.T) {
	terminal, status, err := parsePaymentTerminal([]byte(`{"data":{"terminal":true,"status":"SUCCESS"}}`))
	if err != nil || !terminal || status != "SUCCESS" {
		t.Fatalf("terminal true: %v %v %q", terminal, err, status)
	}

	pending, _, err := parsePaymentTerminal([]byte(`{"data":{"terminal":false,"status":"PENDING"}}`))
	if err != nil || pending {
		t.Fatalf("pending: %v %v", pending, err)
	}

	if _, _, err := parsePaymentTerminal([]byte(`not-json`)); err == nil {
		t.Fatal("malformed JSON must error, not look like a timeout")
	}
	if _, _, err := parsePaymentTerminal([]byte(`{"success":true}`)); err == nil {
		t.Fatal("missing data.terminal must error")
	}
	if _, _, err := parsePaymentTerminal([]byte(`{"data":{"status":"PENDING"}}`)); err == nil {
		t.Fatal("data without terminal must error")
	}
}
