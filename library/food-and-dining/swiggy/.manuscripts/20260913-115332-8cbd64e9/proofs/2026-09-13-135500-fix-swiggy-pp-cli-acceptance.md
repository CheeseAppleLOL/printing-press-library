Acceptance Report: swiggy
  Level: Full Dogfood
  Tests: 168/168 passed (139 skipped as not-applicable: dry-run-only mutating error paths, non-id positionals, etc.)
  Failures: none (final run)
  Fixes applied: 3 (across this phase and the prior local-code-review phase)
    - internal/client/mcp_envelope.go: fixed a systemic bug where tool-level failures were reported as success (nil Go error) — found by Phase 4.95's local code review, re-verified live in this phase.
    - dineout search-restaurants-dineout: added `pp:happy-args` with a real, non-personal query+coordinates (Italian cuisine, Bangalore public coordinates) — the auto-derived generic fixture omitted the required location parameter entirely, which is a real validation error, not a code bug.
    - dineout get-available-slots: added `pp:typed-exit-codes: "0,5"` — this account (confirmed by the user) has never used Dineout, so no real restaurant id is obtainable to fully exercise this command's happy path; the typed-exit-codes annotation accepts the resulting not-found error as an expected outcome, matching the existing precedent for `pay wait` (which has the same "cannot obtain a real in-flight resource to test against" limitation).
  Printing Press issues: 2 (for retro)
    - `generate --force`'s AST-merge does not preserve single-line edits inside DO-NOT-EDIT generated command bodies (only whole added files/functions survive regeneration) — any API needing a hand-written transport-layer patch across many generated command files must reapply that patch after every regen.
    - Parent-shell novel-command files (e.g. order.go, pay.go) populate their `Example:` string once at first generation from research.json and are never refreshed on subsequent regens, unlike the leaf command files which do stay in sync — inconsistent regen behavior within the same "novel scaffold" file class.
  Gate: PASS
