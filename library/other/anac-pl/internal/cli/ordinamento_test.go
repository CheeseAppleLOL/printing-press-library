package cli

import (
	"bytes"
	"testing"
)

func TestWarnOrdinamentoIgnorato(t *testing.T) {
	cases := []struct {
		query, field, dir string
		want              bool
	}{
		{"", "dataPubblicazione", "DESC", false},
		{"ia", "", "", false},
		{"  ", "dataPubblicazione", "", false},
		{"ia", "dataPubblicazione", "DESC", true},
		{"ia", "dataPubblicazione", "", true},
		{"ia", "", "DESC", true},
	}
	for _, c := range cases {
		var b bytes.Buffer
		warnOrdinamentoIgnorato(&b, c.query, c.field, c.dir)
		if got := b.Len() > 0; got != c.want {
			t.Errorf("warnOrdinamentoIgnorato(%q,%q,%q) avviso=%v; want %v", c.query, c.field, c.dir, got, c.want)
		}
	}
}
