package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func TestConValidazioneAvvisi(t *testing.T) {
	var ricevuti map[string]any
	next := func(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		ricevuti = req.GetArguments()
		return mcplib.NewToolResultText("ok"), nil
	}
	h := conValidazioneAvvisi(next)
	call := func(args map[string]any) *mcplib.CallToolResult {
		ricevuti = nil
		req := mcplib.CallToolRequest{}
		req.Params.Arguments = args
		res, _ := h(context.Background(), req)
		return res
	}

	if res := call(map[string]any{"query": "servizi", "scheda": "esiti"}); res.IsError || ricevuti["scheda"] != "7" {
		t.Errorf("scheda esiti: want 7 inoltrato, got %v (errore=%v)", ricevuti, res.IsError)
	}
	for _, args := range []map[string]any{
		{"query": "servizi", "scheda": "AD3"},
		{"query": "servizi", "scheda": "7,8a"},
		{"query": "data visualization", "fuzzy": false},
		{"query": "data visualization", "fuzzy": "false"},
	} {
		if res := call(args); !res.IsError || ricevuti != nil {
			t.Errorf("%v: want errore senza chiamata all'API", args)
		}
	}
	if res := call(map[string]any{"query": "data visualization", "fuzzy": false, "scheda": float64(7)}); res.IsError || ricevuti["scheda"] != "7" {
		t.Errorf("scheda numerica con ricerca esatta: want inoltrata, got %v", ricevuti)
	}
	if res := call(map[string]any{"query": "servizi"}); res.IsError || ricevuti == nil {
		t.Error("senza scheda e ricerca estesa: want inoltrata")
	}
}
