package util

import (
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/pterm/pterm"
)

// FprintJSON marshals v with two-space indent and writes it to w followed by a
// newline. Control runes are escaped rather than replaced, so the document still
// parses into the original values.
func FprintJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, escapeJSONControls(b)) //nolint:forbidigo // escaped above
	return err
}

// escapeJSONControls escapes the runes encoding/json leaves raw (C1, bidi
// overrides, zero-width), which only string contents can hold. Escaped rather
// than replaced, so --json round-trips where the table path loses the value.
func escapeJSONControls(b []byte) string {
	var out []byte
	last := 0
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if keepChar(r, size, true) {
			i += size
			continue
		}
		if out == nil {
			out = make([]byte, 0, len(b))
		}
		out = append(out, b[last:i]...)
		out = appendRuneEscape(out, r)
		i += size
		last = i
	}
	if out == nil {
		return string(b)
	}
	return string(append(out, b[last:]...))
}

// appendRuneEscape appends r as one \uXXXX escape, or as the surrogate pair JSON
// spells a non-BMP rune with.
func appendRuneEscape(dst []byte, r rune) []byte {
	if r <= 0xFFFF {
		return fmt.Appendf(dst, `\u%04x`, r)
	}
	hi, lo := utf16.EncodeRune(r)
	return fmt.Appendf(dst, `\u%04x\u%04x`, hi, lo)
}

// PrintJSON marshals v with two-space indent and prints it to Out followed by a
// newline. Use for single-entity get output and for full (unfiltered) list
// output.
func PrintJSON(v any) error {
	return FprintJSON(Out, v)
}

// PrintJSONColumnFiltered marshals each item, re-decodes it into a map, keeps only
// the requested column keys, and prints the indented array to stdout. Use when the
// caller requested an explicit subset of columns; otherwise use PrintJSON(items).
func PrintJSONColumnFiltered[T any](items []T, columns []string) error {
	filtered, err := columnFilteredMaps(items, columns)
	if err != nil {
		return err
	}
	return PrintJSON(filtered)
}

// columnFilteredMaps round-trips each item through JSON and keeps only the entries
// whose key is in columns. The result preserves item order; per-item maps contain
// only the requested keys that exist on the item.
func columnFilteredMaps[T any](items []T, columns []string) ([]map[string]any, error) {
	filtered := make([]map[string]any, len(items))
	for i := range items {
		filtered[i] = make(map[string]any)
		data, err := json.Marshal(items[i])
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("unmarshaling item: %w", err)
		}
		for _, col := range columns {
			if val, ok := m[col]; ok {
				filtered[i][col] = val
			}
		}
	}
	return filtered, nil
}

// RenderTable is the one path a table takes to the terminal, so callers pass
// raw values. Every cell is sanitized, header included: the stream writer keeps
// \n and \t, and either would forge a row or a column.
func RenderTable(data pterm.TableData) {
	clean := make(pterm.TableData, len(data))
	for i, row := range data {
		clean[i] = make([]string, len(row))
		for j, cell := range row {
			clean[i][j] = Sanitize(cell)
		}
	}
	pterm.DefaultTable.WithHasHeader().WithSeparator(" | ").WithHeaderRowSeparator("").WithData(clean).WithWriter(Out).Render() //nolint:forbidigo // the one sanitized render path
}

// PrintTable renders a header row of column names followed by one row per item.
// valueFn returns the rendered cell string for (item, column) and reports whether
// the column is known; an unknown column yields an "unknown column: %q" error.
// It is generic over the item type so it serves both the api.* entity types and
// the resource package's decryptedResource.
func PrintTable[T any](columns []string, items []T, valueFn func(item T, column string) (string, bool)) error {
	data := pterm.TableData{columns}
	for _, it := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			v, ok := valueFn(it, col)
			if !ok {
				return fmt.Errorf("unknown column: %q", col)
			}
			row[i] = v
		}
		data = append(data, row)
	}
	RenderTable(data)
	return nil
}
