Manifest transcendence rows: 5 planned, 5 built. Phase 3 will not pass until all 5 ship.

## What was built

**Priority 0 (foundation) — generator-emitted, hand-patched for MCP transport:**
- Internal YAML spec authored from the full 51-tool catalog (`research/swiggy-spec.yaml`): 3 resources (food/instamart/dineout), one endpoint per tool, `auth.type: oauth2` scaffold.
- `generate` produced all 51 typed endpoint commands, config/client/doctor/README/SKILL scaffolding, and the MCP server bundle. Generator auto-applied the Cloudflare MCP pattern (code orchestration, hidden endpoint tools, stdio+http transport) since 51 > 50 endpoints.
- **Hand-written transport shim** (`internal/client/mcp_envelope.go`, new file): the generator has zero native support for MCP/JSON-RPC (confirmed by directly researching its spec-format and phase files before this run). This file wraps every generated command's existing `Post`/`PostQueryWithParams` call in the `{"jsonrpc":"2.0","method":"tools/call","params":{"name":...,"arguments":...},"id":...}` envelope and unwraps the response back to the tool's own `{success,data,message}` shape, defensively handling both the bare-envelope response documented on every tool reference page and a full JSON-RPC `result` wrapper in case production behaves differently once real auth is available. JSON-RPC transport errors (`-32001` etc.) are reformatted so the existing `classifyAPIErrorOnly` string-matcher in `helpers.go` still routes them correctly — no new classifier branch needed.
- All 51 generated command files mechanically edited (scripted, not hand-typed) to call `c.MCPToolCall`/`c.MCPToolQuery` with their tool name instead of the raw `Post`/`PostQueryWithParams` calls, preserving every flag, help example, and annotation the generator produced.
- **Hand-written OAuth 2.1 PKCE + Dynamic Client Registration login** (`internal/cli/auth_login.go`, new file; `auth.go` edited to register it): the generator also has no template for this (confirmed via `references/oauth2-pkce-cli-checklist.md`, which is explicitly the hand-write checklist, not a generator feature). Implements every required behavior from that checklist: fresh state + PKCE verifier per attempt, S256 challenge, state validated before code exchange, loopback-only callback listener, separate timeout contexts for the browser wait vs. token exchange, `--json`+`--no-open` refused as an impossible combination, browser launch and network calls skipped under the verify harness, and a token-exchange HTTP client that refuses cross-origin redirects instead of reusing the shared API client's `CheckRedirect`.

**Priority 1 (absorb):** all 51 Swiggy MCP tools absorbed 1:1 as typed commands (see absorb manifest rows 1–51). Every mutating tool has a confirmation-safe `--dry-run` path (generator default) plus the shared MCP envelope wrapper; every read tool routes through `MCPToolQuery`.

**Priority 2 (transcend) — all 5 hand-built and wired:**
1. `history` (`internal/cli/history.go`) — cross-domain spend/order aggregation over live `get_food_orders` + `get_orders` calls (no local cache is enabled — see Data Layer note below).
2. `pay wait` (`internal/cli/pay_wait.go`) — long-polls `check_payment_status` per domain to a terminal state with a configurable `--max-wait`.
3. `order verify-before-retry` (`internal/cli/order_verify_before_retry.go`) — reconciles a failed placement against real order history by amount (+ optional restaurant id) before permitting a retry; explicitly refuses for `--domain dineout` since no orders-list tool exists there.
4. `instamart quick-reorder` (`internal/cli/instamart_quick_reorder.go`) — chains `your_go_to_items → update_cart → checkout` behind a `--confirm` gate; previews without it.
5. `status` (`internal/cli/status.go`) — reads the stored OAuth token's expiry from local config, no network call.

All 5 were auto-scaffolded by the generator from `research.json`'s `novel_features` (including root wiring via `addNovelCommandIfAbsent`) — Priority 2 work was replacing each `TODO: implement novel feature` body with the real implementation, not building Cobra wiring from scratch.

## Generator limitations found (for future printing-press improvement)

- No native MCP/JSON-RPC spec shape or generation path — every JSON-RPC-shaped API will need this same hand-written envelope-wrapper pattern until the generator grows one.
- No native OAuth 2.1 PKCE + Dynamic Client Registration login flow — only `device_code`/`client_credentials`/simple bearer are template-backed today; PKCE+DCR needed the full hand-write per the existing checklist.
- Generator's novel-feature auto-scaffolder correctly detected the `instamart quick-reorder` first-segment collision with the existing `instamart` resource command and printed a warning, but it registered the scaffold under the resource group anyway (`instamart.go`'s `addNovelCommandIfAbsent`) — the warning was more conservative than the actual (correct) behavior.

## What was intentionally deferred

- **No local SQLite cache** (`cache.enabled: false` in the spec): Food/Instamart carts and orders are per-user working state, and enabling a stale pre-read cache over order history didn't meet the Pre-Generation Cache Enrichment bar (no cheap bulk-refresh sync path exists for 51 heterogeneous JSON-RPC tools; building one was out of scope for this run). `history` therefore calls the live API directly every time rather than reading a synced mirror — documented honestly in `history --help` and the `--since` flag's own description.
- Per-domain MCP session-id caching was considered but dropped after re-reading Swiggy's own curl examples: every documented tool call is a single self-contained JSON-RPC POST with just a Bearer header, no separate `initialize` handshake or session-id header shown. The "don't reinitialize per call" guidance in Swiggy's rate-limit docs is interpreted as "don't re-run OAuth/DCR per call" (already satisfied by the 5-day cached access token), not a stateful MCP session requiring its own handshake. This assumption needs empirical confirmation against production once a real OAuth login is available (flagged in the research brief's Reachability Risk section).
- Exact wire response shape for an authenticated tool call (bare tool envelope vs. full JSON-RPC `result` wrapper) is unverified — no real OAuth token was available during this build. `unwrapMCPToolResult` handles both shapes defensively; Phase 5 live dogfood (once the user completes `auth login`) is the first real test of this assumption.

## Skipped body fields

None — every parameter from the source-of-truth tool catalog (fetched from all 51 official reference pages) was carried into the spec's `body:` blocks and is exposed as a generated flag.

## Priority 1 Review Gate

Ran on 3 random Priority 1 commands: `food search-restaurants --help`/`--dry-run`/`--json`, `instamart get-cart --help`/`--dry-run`/`--json`, `dineout book-table --help`/`--dry-run`/`--json`. All three showed realistic examples, dry-run previews without dialing out, and `--json`-shaped output paths. No systemic issue found.

## Phase 3 Completion Gate evidence

- Per-row Cobra resolution: all 5 transcendence commands (`history`, `pay wait`, `order verify-before-retry`, `instamart quick-reorder`, `status`) resolve with `Usage: swiggy-pp-cli <leaf> [flags]` and exit 0.
- `cli-printing-press dogfood --dir <work> --research-dir <run> --json`: `novel_features_check` reports `planned: 5, found: 5`, `missing`/`skipped` absent (both treated as empty/false) — gate jq check passes.
- Dogfood overall verdict: WARN (non-blocking). Issues: 3 pre-existing generator-emitted dead helper functions unrelated to this build (`handleBinaryResponseDelivery`, `paginatedGetWithResponsePath`, `readSecretFromStdin` — narrow API surface never triggers binary responses, response-path pagination, or stdin-secret input), one benign command-depth note (dogfood's matcher noticed both a top-level `status` command and an unrelated pre-existing `auth status` command share the leaf name "status" — both resolve correctly and independently, confirmed above), and a generic "sync uses generic Upsert only" note (no sync/cache path exists in this CLI by design — see deferred section above).
- `go build ./...`, `go vet ./...`, `go test ./...` all pass after every edit in this phase.
