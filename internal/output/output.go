package output

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"
)

type Column struct {
	Key   string
	Label string
}

type TableOptions struct {
	Columns  []string
	NoHeader bool
	Style    string
}

func RenderTable(rows []map[string]string, columns []Column, options TableOptions) (string, error) {
	selected, err := selectColumns(columns, options.Columns)
	if err != nil {
		return "", err
	}
	if options.Style == "" {
		options.Style = "bordered"
	}
	if options.Style != "compact" && options.Style != "plain" && options.Style != "bordered" {
		return "", fmt.Errorf("unsupported table style %q", options.Style)
	}

	widths := tableWidths(rows, selected)
	switch options.Style {
	case "bordered":
		return renderBoxTable(rows, selected, widths, options.NoHeader, lightBox), nil
	case "plain":
		return renderBoxTable(rows, selected, widths, options.NoHeader, asciiBox), nil
	default:
		return renderCompactTable(rows, selected, widths, options.NoHeader), nil
	}
}

func RenderJSON(payload any) (string, error) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

func FilterRows(rows []map[string]string, expression string) ([]map[string]string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return rows, nil
	}

	operator := "="
	parts := strings.SplitN(expression, "!=", 2)
	if len(parts) == 2 {
		operator = "!="
	} else {
		parts = strings.SplitN(expression, "=", 2)
	}
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filter %q: use key=value or key!=value", expression)
	}

	key := strings.TrimSpace(parts[0])
	want := strings.TrimSpace(parts[1])
	if key == "" {
		return nil, fmt.Errorf("invalid filter %q: missing key", expression)
	}

	filtered := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		got := row[key]
		matches := got == want
		if (operator == "=" && matches) || (operator == "!=" && !matches) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func SortRows(rows []map[string]string, expression string) ([]map[string]string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return rows, nil
	}

	descending := strings.HasPrefix(expression, "-")
	key := strings.TrimPrefix(expression, "-")
	if key == "" {
		return nil, fmt.Errorf("invalid sort %q: missing key", expression)
	}

	sorted := slices.Clone(rows)
	slices.SortStableFunc(sorted, func(left, right map[string]string) int {
		result := strings.Compare(left[key], right[key])
		if descending {
			return -result
		}
		return result
	})
	return sorted, nil
}

func selectColumns(columns []Column, requested []string) ([]Column, error) {
	if len(requested) == 0 {
		return columns, nil
	}

	byKey := make(map[string]Column, len(columns))
	for _, column := range columns {
		byKey[column.Key] = column
	}

	selected := make([]Column, 0, len(requested))
	for _, key := range requested {
		column, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("unknown column %q", key)
		}
		selected = append(selected, column)
	}
	return selected, nil
}

func tableWidths(rows []map[string]string, columns []Column) []int {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = displayWidth(col.Label)
		for _, row := range rows {
			if width := displayWidth(row[col.Key]); width > widths[i] {
				widths[i] = width
			}
		}
	}
	return widths
}

func renderCompactTable(rows []map[string]string, columns []Column, widths []int, noHeader bool) string {
	var b strings.Builder
	if !noHeader {
		writeCompactLine(&b, columns, nil, widths, true)
	}
	for _, row := range rows {
		writeCompactLine(&b, columns, row, widths, false)
	}
	return b.String()
}

func writeCompactLine(b *strings.Builder, columns []Column, row map[string]string, widths []int, header bool) {
	for i, col := range columns {
		value := col.Label
		if !header {
			value = row[col.Key]
		}
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(value)
		if i < len(columns)-1 {
			b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(value)))
		}
	}
	b.WriteByte('\n')
}

type boxStyle struct {
	topLeft, topSeparator, topRight          string
	middleLeft, middleSeparator, middleRight string
	bottomLeft, bottomSeparator, bottomRight string
	vertical, horizontal                     string
}

var asciiBox = boxStyle{
	topLeft: "+", topSeparator: "+", topRight: "+",
	middleLeft: "+", middleSeparator: "+", middleRight: "+",
	bottomLeft: "+", bottomSeparator: "+", bottomRight: "+",
	vertical: "|", horizontal: "-",
}

var lightBox = boxStyle{
	topLeft: "┌", topSeparator: "┬", topRight: "┐",
	middleLeft: "├", middleSeparator: "┼", middleRight: "┤",
	bottomLeft: "└", bottomSeparator: "┴", bottomRight: "┘",
	vertical: "│", horizontal: "─",
}

func renderBoxTable(rows []map[string]string, columns []Column, widths []int, noHeader bool, style boxStyle) string {
	var b strings.Builder
	writeBorder(&b, widths, style.topLeft, style.topSeparator, style.topRight, style.horizontal)
	if !noHeader {
		writeBoxLine(&b, columns, nil, widths, true, style.vertical)
		writeBorder(&b, widths, style.middleLeft, style.middleSeparator, style.middleRight, style.horizontal)
	}
	for _, row := range rows {
		writeBoxLine(&b, columns, row, widths, false, style.vertical)
	}
	writeBorder(&b, widths, style.bottomLeft, style.bottomSeparator, style.bottomRight, style.horizontal)
	return b.String()
}

func writeBoxLine(b *strings.Builder, columns []Column, row map[string]string, widths []int, header bool, vertical string) {
	b.WriteString(vertical)
	for i, col := range columns {
		value := col.Label
		if !header {
			value = row[col.Key]
		}
		b.WriteString(" ")
		b.WriteString(value)
		b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(value)))
		b.WriteString(" ")
		b.WriteString(vertical)
	}
	b.WriteByte('\n')
}

func writeBorder(b *strings.Builder, widths []int, left, separator, right, horizontal string) {
	b.WriteString(left)
	for i, width := range widths {
		b.WriteString(strings.Repeat(horizontal, width+2))
		if i == len(widths)-1 {
			b.WriteString(right)
		} else {
			b.WriteString(separator)
		}
	}
	b.WriteByte('\n')
}

func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}
