# Swiggy CLI Polish Report

## Polish skill result (mid-pipeline, forked)

```
Scorecard:        62/100    62/100    +0    (polish's own sandboxed run — see below)
Verify:           94.7%     100%      +5.3%
Tools-audit:      1 pending  0 pending (1 accepted with rationale)
gosec:            2  ->  0
```

**Fixes applied by polish (all verified working after rebuild):**
- Added `pp:typed-exit-codes: "0,2"` to `order.go` and `profile.go` parent-group commands — fixed 2 real verify failures.
- Fixed gosec G112 (Slowloris) in `auth_login.go` by adding `ReadHeaderTimeout` to the loopback OAuth callback `http.Server`.
- Added a narrow, justified `#nosec G204` suppression in `auth_login.go`'s `openBrowser` (fixed per-OS binary, argv-only exec, no shell interpretation).
- Accepted the sole pii-audit finding (`Swiggy Builders Club support`) as `category: api_provider_data` — Swiggy's own published developer-program contact, not customer PII.

**Regression found and fixed in this session (post-polish):** polish also changed `pay.go`'s `pp:typed-exit-codes` from `"0,1"` to `"0,2"`, which was wrong — `pay wait`'s actual timeout/missing-flag error paths return exit 1 (plain `fmt.Errorf`), not 2. This broke 2 live-dogfood tests (168/168 → 166/168). Fixed by widening to `"0,1,2"` and re-verified: full live dogfood matrix back to 168/168, `go test ./...` clean.

## Why polish reported `ship_recommendation: hold`

Polish's own diagnostic run happens in a forked/sandboxed context with no access to this session's real on-disk OAuth credentials (`~/.local/share/swiggy-pp-cli/credentials.toml`) — the same isolation limitation already observed and documented for the Phase 4.85 output-review sub-skill. Its own report is explicit about this: *"Live-check 401s across all 5 samples - no live token in this sandbox, environmental"* and names unpopulated `vision`/`workflows` scorecard fields (a research-phase data gap, not a polish-fixable code issue) as the only other factor holding its sandboxed score below 65.

This session's own `shipcheck` invocation (which correctly threads a real, on-disk OAuth token via `--env-var SWIGGY_ACCESS_TOKEN`) — re-run fresh after every polish fix and after fixing polish's `pay.go` regression — shows:
- **Verdict: PASS (7/7 legs)**
- **Scorecard: 87/100, Grade A, `live_api_verification: 10/10`**
- **Live dogfood (full level): 168/168 passing**

The polish `hold` is a sandbox artifact of the fork's credential isolation, not a finding about the shipped CLI. Confirmed by re-verifying with real production data after incorporating every one of polish's applied fixes.

## Final decision

Per explicit user direction after reviewing this exact discrepancy: fix the regression polish introduced, re-verify with real credentials, and proceed to ship. All of polish's genuinely valuable fixes (gosec security hardening, exit-code corrections for `order`/`profile`) are retained; the one incorrect change (`pay.go`'s exit-code annotation) was corrected.

**Final verdict: ship.**
