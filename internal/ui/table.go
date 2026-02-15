package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Table renders rows of data in aligned columns.
type Table struct {
	w       *tabwriter.Writer
	headers []string
}

// NewTable creates a new table writer with the given column headers.
func NewTable(out io.Writer, headers ...string) *Table {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	t := &Table{w: tw, headers: headers}
	_, _ = fmt.Fprintln(tw, strings.Join(headers, "\t"))
	return t
}

// Row appends a row of values. The number of values should match the number of headers.
func (t *Table) Row(values ...any) {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%v", v)
	}
	_, _ = fmt.Fprintln(t.w, strings.Join(parts, "\t"))
}

// Flush writes the buffered output.
func (t *Table) Flush() error {
	return t.w.Flush()
}
