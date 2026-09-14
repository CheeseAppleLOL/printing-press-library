package cli

import "testing"

func TestParseAffidamentiSort(t *testing.T) {
	cases := []struct {
		field, dir string
		wantField  string
		wantDesc   bool
		wantErr    bool
	}{
		{"", "", "", false, false},
		{"data", "", "data", false, false},
		{"Data", "desc", "data", true, false},
		{"importo", "ASC", "importo", false, false},
		{"", "desc", "", false, true},
		{"dataPubblicazione", "", "", false, true},
		{"data", "giu", "", false, true},
	}
	for _, c := range cases {
		f, d, err := parseAffidamentiSort(c.field, c.dir)
		if (err != nil) != c.wantErr || f != c.wantField || d != c.wantDesc {
			t.Errorf("parseAffidamentiSort(%q,%q) = %q,%v,%v; want %q,%v,err=%v", c.field, c.dir, f, d, err, c.wantField, c.wantDesc, c.wantErr)
		}
	}
}

func TestSortAffidamenti(t *testing.T) {
	rows := func() []affidamentoRow {
		return []affidamentoRow{
			{Data: "2025-03-01", Importo: 50, ImportoNoto: true, CIG: "b"},
			{Data: "", Importo: 10, ImportoNoto: true, CIG: "vuota"},
			{Data: "2024-01-15", Importo: 300, ImportoNoto: true, CIG: "a"},
			{Data: "2025-03-01", Importo: 20, ImportoNoto: true, CIG: "c"},
			{Data: "", Importo: 0, ImportoNoto: false, CIG: "senza"},
		}
	}
	cigs := func(rs []affidamentoRow) string {
		s := ""
		for _, r := range rs {
			s += r.CIG + " "
		}
		return s
	}
	cases := []struct {
		field string
		desc  bool
		want  string
	}{
		{"", false, "b vuota a c senza "},        // nessun ordinamento
		{"data", false, "a b c vuota senza "},    // stabile a parità di data, vuote in fondo
		{"data", true, "b c a vuota senza "},     // vuote in fondo anche in desc
		{"importo", false, "vuota c b a senza "}, // numerico; senza importo in fondo, non prima
		{"importo", true, "a b c vuota senza "},
	}
	for _, c := range cases {
		rs := rows()
		sortAffidamenti(rs, c.field, c.desc)
		if got := cigs(rs); got != c.want {
			t.Errorf("sortAffidamenti(%q, desc=%v) = %q; want %q", c.field, c.desc, got, c.want)
		}
	}
}
