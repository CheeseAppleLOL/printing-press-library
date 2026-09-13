# Swiggy CLI — Novel Features Brainstorm (Pass 1 → Pass 3)

## Pass 1: Customer Models

Four personas grounded in the brief's "Users" and "Product Thesis" sections:

1. **Agent-framework developer** — building a voice/chat commerce agent on top of Builders Club, using the CLI to dry-run flows and debug JSON-RPC/OAuth behavior before wiring a real agent.
2. **Power-user / personal-automation dev** — drives `swiggy-pp-cli` directly from scripts/cron to order food, restock groceries, or book tables without a UI.
3. **Support/ops troubleshooter** — deals with failed or ambiguous orders (5xx during `place_food_order`, stuck UPI confirmations) and needs deterministic diagnosis, not guesswork.
4. **Data-curious self-quantifier** — wants to know what they've spent and ordered across all three independently-siloed Swiggy services, something no single Swiggy surface currently shows.

## Candidates (pre-cut)

| Candidate | One-line pitch | Initial concern |
|---|---|---|
| `swiggy status` | Report auth-token expiry + per-domain session health in one shot | Could overlap with `auth login`; keep read-only |
| `swiggy history` | Cross-domain spend/order aggregation (Food + Instamart + Dineout) from local cache | Must stay local-data aggregation, not a fake API call |
| `swiggy pay wait` | Unified long-poll helper wrapping `check_payment_status` across all 3 servers | Must actually call the real endpoint repeatedly, not simulate |
| `swiggy order verify-before-retry` | Correlate a failed `place_food_order`/`checkout` against `get_food_orders`/`get_orders` before allowing a retry | Matching heuristic must not fabricate data |
| `swiggy address sync` | Copy an address from Food's address book to Instamart's (or vice versa) | On inspection: brief's "shared param shape" note is about internal code reuse, not user-facing demand — weak evidence |
| `swiggy limits` | Local rate-budget tracker against the documented 70/min (30/min write) ceiling | Server doesn't enforce it yet in v1.0 — solves a problem that doesn't bite yet |
| `swiggy instamart quick-reorder` | One command chaining `your_go_to_items → update_cart → checkout` | Must stay one bounded command, not a mini-app |
| `swiggy coupon best` | Recommend best-value coupon for current cart total | Discount-eligibility modeling risks reimplementing business logic the API already owns |

## Survivors and kills

### Killed candidates

| Feature | Kill reason | Closest surviving sibling |
|---|---|---|
| `swiggy address sync` | Brief's "shared param shape" note is implementation-DRY guidance for the generator's own Go code, not evidence of user demand to duplicate addresses across services. Raw 6/10 but evidence doesn't hold up on re-read. | `swiggy status` (both are cross-cutting utility commands) |
| `swiggy limits` | v1.0 doesn't enforce the documented rate ceiling server-side (shed upstream instead); anticipatory value only, duplicates ground the shared error classifier already covers reactively. Raw 6/10. | `swiggy status` |
| `swiggy coupon best` | Modeling "best" coupon risks reimplementing discount-eligibility business logic the API already owns via `apply_coupon`/`apply_food_coupon`. No direct evidence of demand. Raw 4/10, below the ≥5 bar. | `swiggy instamart quick-reorder` |

### Survivors

| # | Feature | Command | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-------|--------------|--------------|----------|------------------|
| 1 | Cross-domain spend/order aggregation | `swiggy history` | 10/10 | hand-code | Joins locally-cached `get_food_orders`, `get_orders` (Instamart), and Dineout booking-status rows in SQLite to compute total spend and order counts across all three servers in one view | Brief: "no shared entities across servers... Food, Instamart, and Dineout do not share carts, orders, sessions, or addresses" (Codebase Intelligence); community precedent `mr-karan/swiggy-analytics` proves demand for order-history analytics per-domain | Use this for a combined view across Food, Instamart, and Dineout. Do NOT use it for single-domain order lookup; use `food get-food-orders` or `instamart get-orders` for that. |
| 2 | Unified UPI long-poll wait | `swiggy pay wait` | 10/10 | hand-code | Repeatedly calls the domain's real `check_payment_status` tool with backoff until a terminal state, instead of the caller hand-rolling the poll loop per server | Brief Top Workflows #4: "long-poll, ~19s server hold, poll until terminal"; Table Stakes: payment-option discovery + confirmation shared across all three servers | Use this after placing an order paid via UPI to block until payment resolves. Do NOT use it for COD orders — call `check-payment-status` once and proceed. |
| 3 | Safe-retry check for non-idempotent order placement | `swiggy order verify-before-retry` | 10/10 | hand-code | Calls `get_food_orders`/`get_orders` and matches on restaurant/amount/time window to determine whether a prior `place_food_order`/`checkout` attempt actually succeeded before allowing a retry | Brief Reachability/Top Workflows #1 (direct quote): "`place_food_order` is explicitly not idempotent... the client must call `get_food_orders` to check whether the order actually placed before ever retrying" | Use this before re-issuing `place-food-order`/`checkout` after a 5xx or timeout. Do NOT use it as a substitute for `get-food-order-details`, which looks up a known order ID directly. |
| 4 | Instamart one-shot reorder | `swiggy instamart quick-reorder` | 10/10 | hand-code | Chains `your_go_to_items → update_cart → checkout` behind a single confirmation prompt for a returning user | Brief Top Workflows #5 (direct quote): "`your_go_to_items` replaces 3-5 search calls for a returning user... A `swiggy instamart quick-reorder` command built on this tool is a strong differentiator" | Use this for a fast repeat purchase of frequently-bought items. Do NOT use it for a first-time or custom order; use `search-products` + `update-cart` + `checkout` for that. |
| 5 | Auth/session health check | `swiggy status` | 8/10 | hand-code | Reads the stored OAuth token's issued/expiry timestamps and each domain's cached MCP session id to report days-remaining-on-token and session-reuse state | Brief Architecture: "Access tokens live 5 days; no refresh-token issuance in v1.0... always treat 401 as re-authenticate"; brief's "do not reconnect/reinitialize per tool call" is called "the single most-repeated operational instruction across every Swiggy doc page" | Use this to check whether you need to re-run `auth login` before starting a session. Do NOT use it to perform login itself — it is read-only. |

All five are additive to the 51 absorbed 1:1 tool commands and the already-absorbed per-domain order-history analytics; none duplicate existing rows in the absorb manifest. This is a first print (`research.json: none`), so no reprint verdicts section applies.
