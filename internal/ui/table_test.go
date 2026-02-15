package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable_render(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "NAME", "VALUE", "OK")
	tbl.Row("alpha", 42, true)
	tbl.Row("beta", 0, false)
	if err := tbl.Flush(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "NAME") {
		t.Errorf("header missing NAME: %q", lines[0])
	}
	if !strings.Contains(lines[1], "alpha") {
		t.Errorf("row 1 missing alpha: %q", lines[1])
	}
}

func TestTable_emptyTable(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "A", "B")
	if err := tbl.Flush(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d", len(lines))
	}
}
