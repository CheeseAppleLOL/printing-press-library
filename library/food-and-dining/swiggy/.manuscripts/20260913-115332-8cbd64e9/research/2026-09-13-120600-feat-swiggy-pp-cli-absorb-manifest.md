# Swiggy CLI Absorb Manifest

## Absorbed (match or beat everything that exists)

All 51 Swiggy MCP tools, absorbed 1:1 as typed commands grouped by server (Food/Instamart/Dineout). Added value is uniform across the set: every command gets the shared JSON-RPC/MCP transport client (persistent session, no per-call reinitialize), the shared `{success,data,message}`/`{success:false,error}` envelope parser and error classifier, `--json`/`--agent`/`--select` output modes, and — for every mutating tool — a confirmation prompt, `--dry-run`, and (where the tool is not idempotent per the docs) a non-idempotency guard.

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | `apply_food_coupon` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food apply-food-coupon` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 2 | `check_payment_status` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food check-payment-status` | Local SQLite cache + offline lookup, shared error classifier |
| 3 | `confirm_order` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food confirm-order` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 4 | `create_address` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food create-address` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 5 | `delete_address` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food delete-address` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 6 | `fetch_food_coupons` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food fetch-food-coupons` | Local SQLite cache + offline lookup, shared error classifier |
| 7 | `flush_food_cart` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food flush-food-cart` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 8 | `get_addresses` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-addresses` | Local SQLite cache + offline lookup, shared error classifier |
| 9 | `get_food_cart` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-food-cart` | Local SQLite cache + offline lookup, shared error classifier |
| 10 | `get_food_delivery_status` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-food-delivery-status` | Local SQLite cache + offline lookup, shared error classifier |
| 11 | `get_food_order_details` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-food-order-details` | Local SQLite cache + offline lookup, shared error classifier |
| 12 | `get_food_orders` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-food-orders` | Local SQLite cache + offline lookup, shared error classifier |
| 13 | `get_payment_options` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-payment-options` | Local SQLite cache + offline lookup, shared error classifier |
| 14 | `get_restaurant_menu` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food get-restaurant-menu` | Local SQLite cache + offline lookup, shared error classifier |
| 15 | `place_food_order` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food place-food-order` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 16 | `report_error` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food report-error` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 17 | `search_menu` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food search-menu` | Local SQLite cache + offline lookup, shared error classifier |
| 18 | `search_restaurants` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food search-restaurants` | Local SQLite cache + offline lookup, shared error classifier |
| 19 | `track_food_order` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food track-food-order` | Local SQLite cache + offline lookup, shared error classifier |
| 20 | `update_food_cart` (food) | Swiggy MCP `food` tool | `swiggy-pp-cli food update-food-cart` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 21 | `apply_coupon` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart apply-coupon` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 22 | `check_payment_status` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart check-payment-status` | Local SQLite cache + offline lookup, shared error classifier |
| 23 | `checkout` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart checkout` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 24 | `clear_cart` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart clear-cart` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 25 | `confirm_order` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart confirm-order` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 26 | `create_address` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart create-address` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 27 | `delete_address` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart delete-address` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 28 | `get_addresses` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-addresses` | Local SQLite cache + offline lookup, shared error classifier |
| 29 | `get_cart` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-cart` | Local SQLite cache + offline lookup, shared error classifier |
| 30 | `get_delivery_status` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-delivery-status` | Local SQLite cache + offline lookup, shared error classifier |
| 31 | `get_order_details` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-order-details` | Local SQLite cache + offline lookup, shared error classifier |
| 32 | `get_orders` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-orders` | Local SQLite cache + offline lookup, shared error classifier |
| 33 | `get_payment_options` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart get-payment-options` | Local SQLite cache + offline lookup, shared error classifier |
| 34 | `list_coupons` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart list-coupons` | Local SQLite cache + offline lookup, shared error classifier |
| 35 | `report_error` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart report-error` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 36 | `search_products` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart search-products` | Local SQLite cache + offline lookup, shared error classifier |
| 37 | `track_order` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart track-order` | Local SQLite cache + offline lookup, shared error classifier |
| 38 | `update_cart` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart update-cart` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 39 | `your_go_to_items` (instamart) | Swiggy MCP `instamart` tool | `swiggy-pp-cli instamart your-go-to-items` | Local SQLite cache + offline lookup, shared error classifier |
| 40 | `book_table` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout book-table` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 41 | `cancel_booking` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout cancel-booking` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 42 | `check_payment_status` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout check-payment-status` | Local SQLite cache + offline lookup, shared error classifier |
| 43 | `confirm_order` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout confirm-order` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 44 | `create_cart` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout create-cart` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 45 | `get_available_slots` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout get-available-slots` | Local SQLite cache + offline lookup, shared error classifier |
| 46 | `get_booking_status` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout get-booking-status` | Local SQLite cache + offline lookup, shared error classifier |
| 47 | `get_payment_options` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout get-payment-options` | Local SQLite cache + offline lookup, shared error classifier |
| 48 | `get_restaurant_details` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout get-restaurant-details` | Local SQLite cache + offline lookup, shared error classifier |
| 49 | `get_saved_locations` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout get-saved-locations` | Local SQLite cache + offline lookup, shared error classifier |
| 50 | `report_error` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout report-error` | Confirmation prompt + `--dry-run`, non-idempotency guard, shared error classifier |
| 51 | `search_restaurants_dineout` (dineout) | Swiggy MCP `dineout` tool | `swiggy-pp-cli dineout search-restaurants-dineout` | Local SQLite cache + offline lookup, shared error classifier |
| 52 | Most-ordered dish / monthly spend / weekday order distribution | `mr-karan/swiggy-analytics` (unofficial community CLI) | `(behavior in swiggy-pp-cli history)` — local-SQLite aggregation over cached `get_food_orders`/`get_orders` rows | Same insight as the community tool, but spans Food + Instamart in one command instead of Food-only, and needs no separate OTP-scraping login — reuses the CLI's own OAuth session |
| 53 | Order via terminal | `swiggy-order` (PyPI) | `(generated endpoint) food place-food-order` / `(generated endpoint) instamart checkout` | Real OAuth-backed official API call (not a scraped/reverse-engineered session), plus confirmation prompt and non-idempotency guard the PyPI tool does not have |

## Transcendence (only possible with our approach)

Five features survived a three-pass brainstorm (customer model → 13 candidates → adversarial cut) run via subagent. Full audit trail (personas, all candidates, kill reasons) archived at `2026-09-13-120500-novel-features-brainstorm.md`.

| # | Feature | Command | Score | Buildability | How It Works | Evidence | Long Description |
|---|---------|---------|-------|--------------|--------------|----------|------------------|
| 1 | Cross-domain spend/order aggregation | `history` | 10/10 | hand-code | Joins locally-cached `get_food_orders`, `get_orders` (Instamart), and Dineout booking-status rows in SQLite to compute total spend and order counts across all three servers in one view | Brief: "Food, Instamart, and Dineout do not share carts, orders, sessions, or addresses"; community precedent `mr-karan/swiggy-analytics` proves demand for order-history analytics | Use this for a combined view across Food, Instamart, and Dineout. Do NOT use it for single-domain order lookup; use `food get-food-orders` or `instamart get-orders` for that. |
| 2 | Unified UPI long-poll wait | `pay wait` | 10/10 | hand-code | Repeatedly calls the domain's real `check_payment_status` tool with backoff until a terminal state, instead of the caller hand-rolling the poll loop per server | Brief Top Workflows #4: "long-poll, ~19s server hold, poll until terminal" | Use this after placing an order paid via UPI to block until payment resolves. Do NOT use it for COD orders — call `check-payment-status` once and proceed. |
| 3 | Safe-retry check for non-idempotent order placement | `order verify-before-retry` | 10/10 | hand-code | Calls `get_food_orders`/`get_orders` and matches on restaurant/amount/time window to determine whether a prior `place_food_order`/`checkout` attempt actually succeeded before allowing a retry | Brief (direct quote): "`place_food_order` is explicitly not idempotent... the client must call `get_food_orders` to check whether the order actually placed before ever retrying" | Use this before re-issuing `place-food-order`/`checkout` after a 5xx or timeout. Do NOT use it as a substitute for `get-food-order-details`, which looks up a known order ID directly. |
| 4 | Instamart one-shot reorder | `instamart quick-reorder` | 10/10 | hand-code | Chains `your_go_to_items → update_cart → checkout` behind a single confirmation prompt for a returning user | Brief Top Workflows #5 (direct quote): "`your_go_to_items` replaces 3-5 search calls for a returning user... A `swiggy instamart quick-reorder` command built on this tool is a strong differentiator" | Use this for a fast repeat purchase of frequently-bought items. Do NOT use it for a first-time or custom order; use `search-products` + `update-cart` + `checkout` for that. |
| 5 | Auth/session health check | `status` | 8/10 | hand-code | Reads the stored OAuth token's issued/expiry timestamps and each domain's cached MCP session id to report days-remaining-on-token and session-reuse state | Brief: "Access tokens live 5 days; no refresh-token issuance in v1.0"; "do not reconnect/reinitialize per tool call" called "the single most-repeated operational instruction across every Swiggy doc page" | Use this to check whether you need to re-run `auth login` before starting a session. Do NOT use it to perform login itself — it is read-only. |

### Killed candidates (from the brainstorm, kept for audit trail)

| Feature | Kill reason | Closest surviving sibling |
|---|---|---|
| `address sync` | "Shared param shape" evidence in the brief is implementation-DRY guidance for the generator's own Go code, not user demand to duplicate addresses across services | `status` |
| `limits` | v1.0 doesn't enforce the documented rate ceiling server-side yet; anticipatory value only | `status` |
| `coupon best` | Modeling "best" coupon risks reimplementing discount-eligibility logic the API already owns | `instamart quick-reorder` |

No `research.json` reprint verdicts section — this is a first print.
