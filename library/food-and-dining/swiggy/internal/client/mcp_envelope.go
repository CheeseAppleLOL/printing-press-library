// Copyright 2026 Som Samantray and contributors. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// Swiggy MCP is a genuine Model Context Protocol server (JSON-RPC 2.0 over
// HTTP, one POST endpoint per domain), not a REST API. Every tool call must
// be wrapped in a `tools/call` JSON-RPC envelope; every response must be
// unwrapped back to the tool's own {success,data,message} shape before the
// generated command layer (which expects a flat JSON body) touches it.
//
// This file is hand-written (not generator-emitted) because cli-printing-press
// has no native MCP/JSON-RPC support: generate --spec assumes one HTTP verb +
// path = one REST operation. The generated per-tool command files already
// call c.Post / c.PostWithParams / c.PostQueryWithParams with the tool's
// arguments as the body; MCPToolCall and MCPToolQuery below sit in front of
// those unmodified generated methods and do the envelope wrap/unwrap, so all
// of the generated retry/rate-limit/cache/error machinery in client.go keeps
// working unchanged.

var mcpRequestID int64

func nextMCPRequestID() int64 {
	return atomic.AddInt64(&mcpRequestID, 1)
}

// jsonRPCRequest is the envelope every Swiggy MCP tool call must be wrapped
// in. See https://mcp.swiggy.com/builders/docs/start/authenticate/ and any
// tool reference page's curl example.
type jsonRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  jsonRPCCallOne `json:"params"`
	ID      int64          `json:"id"`
}

type jsonRPCCallOne struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// wrapMCPToolCall builds the tools/call JSON-RPC envelope for one tool
// invocation. arguments is normally the map[string]any the generated command
// already built from its flags (or the --stdin JSON body); it becomes
// params.arguments verbatim.
func wrapMCPToolCall(toolName string, arguments any) jsonRPCRequest {
	if arguments == nil {
		arguments = map[string]any{}
	}
	return jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  jsonRPCCallOne{Name: toolName, Arguments: arguments},
		ID:      nextMCPRequestID(),
	}
}

// jsonRPCTransportError mirrors the shape of a JSON-RPC 2.0 error object.
// Swiggy's docs (reference/errors.md) document -32001 for unauthenticated/
// expired sessions and -32603 for unexpected internal failures at this
// transport layer, distinct from the tool-level {success:false,error:{...}}
// domain-failure envelope.
type jsonRPCTransportError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCEnvelopeShape struct {
	JSONRPC string                 `json:"jsonrpc"`
	Result  json.RawMessage        `json:"result"`
	Error   *jsonRPCTransportError `json:"error"`
}

// mcpResultShape is the real production shape of a Swiggy MCP tool response,
// confirmed empirically against https://mcp.swiggy.com/food with a live
// OAuth token (2026-09-13): a full MCP JSON-RPC response wrapping a
// human-prose `content[0].text` block PLUS a separate `structuredContent`
// object carrying the actual machine-readable payload. This directly
// contradicts every documented example on mcp.swiggy.com/builders/docs,
// which shows a bare {success,data,message} envelope with no outer
// jsonrpc/result/structuredContent wrapper at all — the docs describe the
// tool's logical contract, not the real wire shape. Recorded in the research
// brief's Reachability Risk section.
type mcpResultShape struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
	IsError           bool            `json:"isError"`
}

// unwrapMCPToolResult converts a raw HTTP response body into a
// {"success":true,"data":<payload>} envelope — the shape every generated
// command and every hand-written novel command (history.go,
// order_verify_before_retry.go, etc.) already expects to parse, matching
// what Swiggy's own docs promise even though the real wire shape is
// different underneath.
//
// Handles the shapes actually observed or plausible on this wire:
//  1. Full JSON-RPC response with `structuredContent` (confirmed production
//     shape): {"result":{"content":[{"type":"text","text":"<prose for an
//     LLM>"}],"structuredContent":{...actual data...}}}. structuredContent
//     is promoted to `data`; the prose in content[] is discarded (it is
//     written for an LLM's judgment, e.g. "Ask the user to confirm...", not
//     for a CLI's structured output).
//  2. Full JSON-RPC response with no structuredContent, only a text content
//     block that happens to parse as JSON: falls back to parsing
//     content[0].text as the data.
//  3. Full JSON-RPC response with no content/structuredContent at all:
//     `result` itself is treated as the data (e.g. {"result":{"success":
//     true,"data":{...}}} if a future server version matches the docs
//     literally).
//  4. The bare tool envelope directly, no outer jsonrpc/result wrapper
//     (what every documented example on mcp.swiggy.com/builders/docs shows):
//     passed through unchanged.
//  5. A JSON-RPC transport-level error ({"error":{"code":-32001,...}}) is
//     converted into a Go error whose message contains an "HTTP nnn" token so
//     the existing classifyAPIErrorOnly string-matching in internal/cli
//     (helpers.go) still classifies it correctly (auth/rate-limit/etc)
//     without needing its own new switch arm.
func unwrapMCPToolResult(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var probe jsonRPCEnvelopeShape
	if err := json.Unmarshal(raw, &probe); err != nil {
		// Not a JSON object we recognize (e.g. a bare array or scalar) — hand
		// it back unchanged and let the caller's own JSON handling decide.
		return raw, nil
	}
	if probe.Error != nil {
		return nil, jsonRPCErrorToClassifiableError(probe.Error)
	}
	if probe.JSONRPC == "" && len(probe.Result) == 0 {
		// No jsonrpc/result envelope present at all: shape 4, the bare tool
		// envelope. Every generated command's error handling hinges on the
		// (data, status, err) triple returned by MCPToolCall/MCPToolQuery: it
		// only calls classifyAPIError when err != nil, and otherwise reports
		// success based on the HTTP status code alone (see e.g.
		// instamart_checkout.go). A bare {"success":false,...} tool envelope
		// carries no non-2xx HTTP status, so without this check a logical
		// tool failure would silently render as a reported success across
		// every call site — including safety-critical ones like
		// instamart_quick_reorder.go's update_cart step. Surface it as a real
		// Go error here so the existing err != nil handling in every caller
		// (generated and hand-written) catches it uniformly.
		if toolErr := bareToolEnvelopeError(raw); toolErr != nil {
			return nil, toolErr
		}
		return raw, nil
	}

	var payload json.RawMessage
	var isError bool
	if len(probe.Result) > 0 {
		var result mcpResultShape
		if err := json.Unmarshal(probe.Result, &result); err == nil {
			isError = result.IsError
			switch {
			case len(result.StructuredContent) > 0:
				payload = result.StructuredContent
			case len(result.Content) > 0:
				// Best-effort: the text block is usually LLM-facing prose, not
				// JSON. Only use it if it actually parses as JSON; otherwise
				// wrap it as a message string so the caller still gets valid
				// JSON instead of a parse error.
				text := result.Content[0].Text
				if json.Valid([]byte(text)) {
					payload = json.RawMessage(text)
				} else {
					msg, _ := json.Marshal(text)
					payload = json.RawMessage(`{"message":` + string(msg) + `}`)
				}
			default:
				payload = probe.Result
			}
		} else {
			payload = probe.Result
		}
	} else {
		payload = raw
	}

	envelope := struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data,omitempty"`
		Error   json.RawMessage `json:"error,omitempty"`
	}{Success: !isError}
	if isError {
		envelope.Error = payload
	} else {
		envelope.Data = payload
	}
	out, err := json.Marshal(envelope)
	if err != nil {
		return payload, nil
	}
	if isError {
		// Same reasoning as the shape-4 case above: a tool-level failure
		// (result.isError) must become a real Go error, not just a
		// success:false field buried in the returned body, or every caller
		// that only checks err != nil will report success anyway.
		return nil, mcpToolErrorFromPayload(payload)
	}
	return out, nil
}

// bareToolEnvelopeError inspects the bare (no jsonrpc/result wrapper) tool
// envelope documented on mcp.swiggy.com/builders/docs for an explicit
// "success":false, converting it into a classifiable Go error. Returns nil
// when the envelope reports success (or omits the field entirely — some
// tools may only ever emit {"data":...} on success).
func bareToolEnvelopeError(raw json.RawMessage) error {
	var probe struct {
		Success *bool           `json:"success"`
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil
	}
	if probe.Success == nil || *probe.Success {
		return nil
	}
	payload := probe.Error
	if len(payload) == 0 && probe.Message != "" {
		payload, _ = json.Marshal(probe.Message)
	}
	return mcpToolErrorFromPayload(payload)
}

// mcpToolErrorFromPayload formats a tool-level failure payload (the `error`
// field of a {"success":false,"error":...} envelope, or the isError content
// block) as a Go error whose message contains an "HTTP nnn" token — matching
// the convention jsonRPCErrorToClassifiableError uses — so the existing
// classifyAPIErrorOnly string-matching in internal/cli/helpers.go still
// routes it to a sensible exit code/hint without a bespoke switch arm for
// this transport.
func mcpToolErrorFromPayload(payload json.RawMessage) error {
	return fmt.Errorf("POST returned HTTP 500: %s (mcp tool error)", mcpToolErrorMessage(payload))
}

func mcpToolErrorMessage(payload json.RawMessage) string {
	if len(payload) == 0 {
		return "tool reported failure with no error details"
	}
	var obj struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(payload, &obj) == nil && obj.Message != "" {
		return obj.Message
	}
	var s string
	if json.Unmarshal(payload, &s) == nil && s != "" {
		return s
	}
	return string(payload)
}

// jsonRPCErrorToClassifiableError formats a JSON-RPC transport error so the
// existing HTTP-status-string classifier in internal/cli/helpers.go
// (classifyAPIErrorOnly) still routes it to the right exit code and hint
// text. -32001 (unauthenticated/expired session per Swiggy's error docs)
// maps to the same treatment as a real HTTP 401; anything else maps to 500.
func jsonRPCErrorToClassifiableError(rpcErr *jsonRPCTransportError) error {
	httpLike := 500
	if rpcErr.Code == -32001 {
		httpLike = 401
	}
	return fmt.Errorf("POST returned HTTP %d: %s (json-rpc code %d)", httpLike, rpcErr.Message, rpcErr.Code)
}

// MCPToolCall issues a mutating (write) Swiggy MCP tool call: wraps arguments
// in the tools/call envelope, POSTs through the generated client's existing
// Post (full retry/rate-limit/cache-invalidation machinery unchanged), then
// unwraps the response back to the tool's own envelope.
//
// mcpTransportHeaders sets Accept to both application/json and
// text/event-stream. Confirmed required against production: an initial call
// with only application/json (the generated client's default) returned
// HTTP 406 with {"error":{"code":-32000,"message":"Not Acceptable: Client
// must accept both application/json and text/event-stream"}} — Swiggy MCP
// enforces the full MCP Streamable HTTP transport's Accept contract even
// though every tool reference page's example curl omits it.
func mcpTransportHeaders() map[string]string {
	return map[string]string{"Accept": "application/json, text/event-stream"}
}

func (c *Client) MCPToolCall(ctx context.Context, path, toolName string, arguments any) (json.RawMessage, int, error) {
	envelope := wrapMCPToolCall(toolName, arguments)
	raw, status, err := c.PostWithHeaders(ctx, path, envelope, mcpTransportHeaders())
	if err != nil {
		return nil, status, err
	}
	data, err := unwrapMCPToolResult(raw)
	return data, status, err
}

// MCPToolQuery issues a read-only Swiggy MCP tool call. Swiggy's search/get/
// list tools ride a JSON-RPC POST on the wire but do not mutate remote
// state, so this routes through the generated client's PostQueryWithParamsAndHeaders
// (doRead path) instead of Post — matching the existing mcp:read-only
// annotation the generator already places on these commands, and keeping the
// verify-mode short-circuit gate correctly scoped to real mutations only.
func (c *Client) MCPToolQuery(ctx context.Context, path, toolName string, params map[string]string, arguments any) (json.RawMessage, int, error) {
	envelope := wrapMCPToolCall(toolName, arguments)
	raw, status, err := c.PostQueryWithParamsAndHeaders(ctx, path, params, envelope, mcpTransportHeaders())
	if err != nil {
		return nil, status, err
	}
	data, err := unwrapMCPToolResult(raw)
	return data, status, err
}
