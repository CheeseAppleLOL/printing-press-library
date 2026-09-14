// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package mcp

import (
	"context"
	"fmt"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mvanhorn/printing-press-library/library/other/anac-pl/internal/cli"
)

// conValidazioneAvvisi applica al tool avvisi_search le stesse verifiche di
// `avvisi search`: l'handler generato chiama l'API direttamente, e senza questo
// passaggio `scheda` come nome ("esiti"), un codice dei risultati ("AD3") o la
// ricerca esatta senza tipologia arrivavano al servizio, che risponde 500.
func conValidazioneAvvisi(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		args := req.GetArguments()
		scheda := ""
		if v, ok := args["scheda"]; ok && v != nil {
			scheda = fmt.Sprint(v)
		}
		fuzzy := true
		switch v := args["fuzzy"].(type) {
		case bool:
			fuzzy = v
		case string:
			fuzzy = !strings.EqualFold(strings.TrimSpace(v), "false")
		}
		tpl, err := cli.ValidaRicercaAvvisi(scheda, fuzzy)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		if tpl != "" {
			args["scheda"] = tpl
		}
		return next(ctx, req)
	}
}
