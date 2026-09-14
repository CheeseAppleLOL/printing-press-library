package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/other/ars-sicilia/internal/store"
	"github.com/spf13/cobra"
)

func runStaleOn(t *testing.T, dbPath string) ([]staleEntry, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	err := runSyncStale(cmd, &rootFlags{asJSON: true}, dbPath, "7d")
	if err != nil {
		return nil, err
	}
	var entries []staleEntry
	if jerr := json.Unmarshal(out.Bytes(), &entries); jerr != nil {
		t.Fatalf("output non JSON: %v\n%s", jerr, out.String())
	}
	return entries, nil
}

// A store that cannot be read must not look like "never synced".
func TestSyncStaleMissingOrEmptyDBNamesThePath(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "store.db")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(dir, "missing.db"), empty} {
		entries, err := runStaleOn(t, p)
		if err != nil {
			t.Fatalf("%s: errore inatteso: %v", p, err)
		}
		if len(entries) == 0 {
			t.Fatalf("%s: report vuoto", p)
		}
		for _, e := range entries {
			if !strings.Contains(e.Hint, p) || strings.Contains(e.Hint, "Mai sincronizzato") {
				t.Fatalf("%s: hint %q deve citare il percorso e non dire «Mai sincronizzato»", p, e.Hint)
			}
		}
	}
}

func TestSyncStaleUnreadableDBIsAnError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "garbage.db")
	if err := os.WriteFile(p, []byte("non un database sqlite, solo testo di prova abbastanza lungo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runStaleOn(t, p); err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("atteso errore che citi %s, ottenuto %v", p, err)
	}
}

func TestSyncStaleHealthyDB(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data.db")
	db, err := store.OpenWithContext(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Upsert("ddl", "18-1", json.RawMessage(`{"id":"18-1"}`)); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSyncState("ddl", "", 1); err != nil {
		t.Fatal(err)
	}
	db.Close()

	entries, err := runStaleOn(t, p)
	if err != nil {
		t.Fatal(err)
	}
	var ddl, other *staleEntry
	for i := range entries {
		switch entries[i].Archivio {
		case "ddl":
			ddl = &entries[i]
		case "mozioni":
			other = &entries[i]
		}
	}
	if ddl == nil || ddl.Records != 1 || ddl.LastSync == "" || ddl.Stale {
		t.Fatalf("ddl: %+v", ddl)
	}
	if other == nil || other.Records != 0 || !strings.Contains(other.Hint, "Mai sincronizzato") {
		t.Fatalf("mozioni: %+v", other)
	}
}

// A stat failure other than "does not exist" is an error, not a missing store.
func TestSyncStaleInaccessiblePathIsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "data.db")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := runStaleOn(t, p); err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("atteso errore che citi %s, ottenuto %v", p, err)
	}
}
