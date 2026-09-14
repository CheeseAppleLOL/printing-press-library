// Copyright 2026 aborruso. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
)

// risolviScheda converte il valore di --scheda nel numero template che la
// ricerca full-text accetta. Verificato il 13/09/2026 sul traffico del form del
// portale e via API: il servizio vuole il template (7 per gli esiti), risponde
// 500 ai codici che compaiono nei risultati (AD3, A1_29) e, con più valori
// separati da virgola, usa solo il primo senza avvisare. Nomi e slug di
// 'tipologie list' vengono accettati come in `cerca --tipologia`.
func risolviScheda(scheda string) (string, error) {
	if strings.TrimSpace(scheda) == "" {
		return "", nil
	}
	if strings.Contains(scheda, ",") {
		return "", fmt.Errorf("--scheda accetta un solo valore: con più valori separati da virgola la ricerca di ANAC usa solo il primo")
	}
	tpl, ok := resolveTipologia(scheda)
	if !ok {
		return "", fmt.Errorf("--scheda %q non riconosciuta: serve il numero template o il nome di 'tipologie list' (es. 4 o bandi, 7 o esiti). I codici dei risultati come AD3 non sono accettati dal servizio", scheda)
	}
	return tpl, nil
}

// ValidaRicercaAvvisi applica a --scheda e alla ricerca esatta le verifiche di
// `avvisi search` e restituisce il template da inviare. La usa anche il tool
// MCP avvisi_search, che chiama l'API senza passare dal comando.
func ValidaRicercaAvvisi(scheda string, fuzzy bool) (string, error) {
	tpl, err := risolviScheda(scheda)
	if err != nil {
		return "", err
	}
	return tpl, verificaRicercaEsatta(fuzzy, tpl)
}

// verificaRicercaEsatta blocca la ricerca esatta senza tipologia: ANAC risponde
// 500 ("Index: 0") con o senza parole chiave. Il portale non ci arriva perché
// nel form la tipologia è obbligatoria.
func verificaRicercaEsatta(fuzzy bool, codiceScheda string) error {
	if !fuzzy && strings.TrimSpace(codiceScheda) == "" {
		return fmt.Errorf("la ricerca esatta richiede una tipologia (--tipologia in 'cerca', --scheda in 'avvisi search'): senza, il servizio risponde 500")
	}
	return nil
}
