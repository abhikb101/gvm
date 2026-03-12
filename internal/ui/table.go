package ui

import (
	"fmt"
	"strings"
)

// Table renders aligned columns of data.
type Table struct {
	rows   [][]string
	widths []int
}

// NewTable creates a new Table.
func NewTable() *Table {
	return &Table{}
}

// AddRow appends a row of columns.
func (t *Table) AddRow(cols ...string) {
	t.rows = append(t.rows, cols)
	for i, col := range cols {
		// Strip ANSI escape codes for width calculation
		plainLen := len(stripANSI(col))
		if i >= len(t.widths) {
			t.widths = append(t.widths, plainLen)
		} else if plainLen > t.widths[i] {
			t.widths[i] = plainLen
		}
	}
}

// Render prints the table with left-aligned columns.
func (t *Table) Render() {
	for _, row := range t.rows {
		var parts []string
		for i, col := range row {
			if i < len(t.widths)-1 {
				// Pad with spaces, accounting for ANSI escape codes
				plain := stripANSI(col)
				padding := t.widths[i] - len(plain)
				if padding < 0 {
					padding = 0
				}
				parts = append(parts, col+strings.Repeat(" ", padding))
			} else {
				parts = append(parts, col)
			}
		}
		fmt.Println(strings.Join(parts, "  "))
	}
}

func stripANSI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			// Skip until 'm' (end of ANSI escape)
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip the 'm'
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
