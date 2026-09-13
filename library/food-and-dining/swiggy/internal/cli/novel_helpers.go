// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-written shared helpers for the novel (transcendence) commands. Not
// generator-emitted; safe to edit and extend.

package cli

// swiggyDomainPath maps a --domain flag value to its MCP server path. Swiggy
// Food, Instamart, and Dineout are three independent MCP servers with no
// shared carts, orders, sessions, or addresses (see the research brief's
// Codebase Intelligence section) but the same shared tools (check_payment_status,
// get_payment_options, confirm_order, report_error) exist verbatim on each,
// so novel commands that operate "per domain" key off this map rather than
// hardcoding one server.
func swiggyDomainPath(domain string) (string, bool) {
	switch domain {
	case "food":
		return "/food", true
	case "instamart":
		return "/im", true
	case "dineout":
		return "/dineout", true
	default:
		return "", false
	}
}
