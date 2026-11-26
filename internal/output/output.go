/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package output renders CTyun command results as stable-key tables or JSON.
package output

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/mattn/go-runewidth"
)

// Column describes one stable output column and its display label.
type Column struct {
	Key   string
	Label string
}

// TableOptions controls table column selection, headers, and visual style.
type TableOptions struct {
	Columns    []string
	NoHeader   bool
	Style      string
	Vertical   bool
	FieldLabel string
	ValueLabel string
	MaxWidth   int
}

// RenderTable formats rows with stable column keys and localized labels.
func RenderTable(rows []map[string]string, columns []Column, options TableOptions) (string, error) {
	selectedKeys, err := ResolveColumnSelectors(columns, options.Columns)
	if err != nil {
		return "", err
	}
	options.Columns = selectedKeys
	selected := selectColumns(columns, options.Columns)
	if options.Vertical {
		rows, selected = verticalRows(rows, selected, options)
		options.Columns = nil
	}
	if options.Style == "" {
		options.Style = "bordered"
	}
	if options.Style != "compact" && options.Style != "plain" && options.Style != "bordered" {
		return "", diagnostic.New("error.unsupported_table_style", options.Style)
	}

	widths := constrainTableWidths(tableWidths(rows, selected), options.Style, options.MaxWidth)
	switch options.Style {
	case "bordered":
		return renderBoxTable(rows, selected, widths, options.NoHeader, lightBox), nil
	case "plain":
		return renderBoxTable(rows, selected, widths, options.NoHeader, asciiBox), nil
	default:
		return renderCompactTable(rows, selected, widths, options.NoHeader), nil
	}
}

// verticalRows converts a single object row into field/value rows.
func verticalRows(rows []map[string]string, columns []Column, options TableOptions) ([]map[string]string, []Column) {
	fieldLabel := options.FieldLabel
	if fieldLabel == "" {
		fieldLabel = "Field"
	}
	valueLabel := options.ValueLabel
	if valueLabel == "" {
		valueLabel = "Value"
	}
	verticalColumns := []Column{
		{Key: "field", Label: fieldLabel},
		{Key: "value", Label: valueLabel},
	}
	if len(rows) != 1 {
		return rows, columns
	}
	row := rows[0]
	vertical := make([]map[string]string, 0, len(columns))
	for _, column := range columns {
		vertical = append(vertical, map[string]string{
			"field": column.Label,
			"value": row[column.Key],
		})
	}
	return vertical, verticalColumns
}

// RenderJSON pretty-prints payload without changing the original CTyun JSON
// shape.
func RenderJSON(payload any) (string, error) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded) + "\n", nil
}

// FilterRows applies a key=value or key!=value filter using stable table keys.
func FilterRows(rows []map[string]string, expression string) ([]map[string]string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return rows, nil
	}

	key, operator, want, err := parseFilterExpression(expression)
	if err != nil {
		return nil, err
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

// SortRows orders rows by a stable table key, using a leading "-" for
// descending order.
func SortRows(rows []map[string]string, expression string) ([]map[string]string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return rows, nil
	}

	descending := strings.HasPrefix(expression, "-")
	key := strings.TrimPrefix(expression, "-")
	if key == "" {
		return nil, diagnostic.New("error.invalid_sort_missing_key", expression)
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

// ResolveColumnSelectors maps user-provided column keys or visible labels to
// stable table keys.
func ResolveColumnSelectors(columns []Column, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return requested, nil
	}
	resolved := make([]string, 0, len(requested))
	for _, list := range requested {
		for text := strings.TrimSpace(list); text != ""; {
			item, rest, more := strings.Cut(text, ",")
			selector := strings.TrimSpace(item)
			key, ok := resolveColumnSelector(columns, selector)
			if more {
				if compound, suffix := matchCompoundLabel(columns, text, ","); compound != "" {
					key, ok, rest = compound, true, strings.TrimPrefix(suffix, ",")
				}
			}
			if ok || selector != "" {
				if !ok {
					return nil, diagnostic.New("error.unknown_column", selector)
				}
				resolved = append(resolved, key)
			}
			text = strings.TrimSpace(rest)
		}
	}
	return resolved, nil
}

// ResolveFilterExpression maps a filter key or visible column label to a
// stable table key while preserving the operator and value.
func ResolveFilterExpression(columns []Column, expression string) (string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "", nil
	}
	key, operator, value, err := parseFilterExpression(expression)
	resolved, ok := resolveColumnSelector(columns, key)
	if !ok || resolved != key {
		if compound, suffix := matchCompoundLabel(columns, expression, "="); compound != "" && suffix != "" {
			return compound + suffix, nil
		}
	}
	if err != nil {
		return "", err
	}
	if !ok {
		return "", diagnostic.New("error.unknown_filter_key", key)
	}
	return resolved + operator + value, nil
}

// matchCompoundLabel finds the longest visible label containing an option delimiter,
// leaving the next list separator or comparison operator for the caller.
func matchCompoundLabel(columns []Column, text, delimiter string) (key, suffix string) {
	longest := 0
	for _, column := range columns {
		if len(column.Label) <= longest || !strings.Contains(column.Label, delimiter) {
			continue
		}
		rest, ok := strings.CutPrefix(text, column.Label)
		rest = strings.TrimSpace(rest)
		if ok && (rest == "" || strings.HasPrefix(rest, delimiter) || delimiter == "=" && strings.HasPrefix(rest, "!=")) {
			if delimiter == "," && isColumnList(columns, column.Label) {
				continue
			}
			longest, key, suffix = len(column.Label), column.Key, rest
		}
	}
	return key, suffix
}

// isColumnList reports whether reading a compound label as separate selectors already succeeds.
// An existing column list keeps its meaning when a new label has the same spelling.
func isColumnList(columns []Column, label string) bool {
	for part := range strings.SplitSeq(label, ",") {
		if _, ok := resolveColumnSelector(columns, part); !ok {
			return false
		}
	}
	return true
}

// parseFilterExpression separates the first comparison operator from its value,
// preserving any additional equals signs or inequality syntax in the value.
func parseFilterExpression(expression string) (key, operator, value string, err error) {
	key, value, found := strings.Cut(expression, "=")
	if !found {
		return "", "", "", diagnostic.New("error.invalid_filter_syntax", expression)
	}
	operator = "="
	if before, inequality := strings.CutSuffix(key, "!"); inequality {
		key = before
		operator = "!="
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", "", diagnostic.New("error.invalid_filter_missing_key", expression)
	}
	return key, operator, strings.TrimSpace(value), nil
}

// ResolveSortExpression maps a sort key or visible column label to a stable
// table key while preserving descending order.
func ResolveSortExpression(columns []Column, expression string) (string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "", nil
	}
	descending := strings.HasPrefix(expression, "-")
	key := strings.TrimSpace(strings.TrimPrefix(expression, "-"))
	if key == "" {
		return "", diagnostic.New("error.invalid_sort_missing_key", expression)
	}
	resolved, ok := resolveColumnSelector(columns, key)
	if !ok {
		return "", diagnostic.New("error.unknown_sort_key", key)
	}
	if descending {
		return "-" + resolved, nil
	}
	return resolved, nil
}

// resolveColumnSelector resolves a stable key first, then the visible label.
func resolveColumnSelector(columns []Column, selector string) (string, bool) {
	selector = strings.TrimSpace(selector)
	for _, column := range columns {
		if selector == column.Key {
			return column.Key, true
		}
	}
	for _, column := range columns {
		if selector == column.Label {
			return column.Key, true
		}
	}
	return "", false
}

// selectColumns resolves requested stable keys to display columns.
func selectColumns(columns []Column, requested []string) []Column {
	if len(requested) == 0 {
		return columns
	}

	byKey := make(map[string]Column, len(columns))
	for _, column := range columns {
		byKey[column.Key] = column
	}

	selected := make([]Column, 0, len(requested))
	for _, key := range requested {
		selected = append(selected, byKey[key])
	}
	return selected
}

// tableWidths fits each column to the widest physical line in its label or values.
func tableWidths(rows []map[string]string, columns []Column) []int {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = widestCellLine(col.Label)
		for _, row := range rows {
			widths[i] = max(widths[i], widestCellLine(row[col.Key]))
		}
	}
	return widths
}

// widestCellLine measures each newline-delimited line independently.
func widestCellLine(cell string) int {
	width := 0
	for line := range strings.SplitSeq(cell, "\n") {
		width = max(width, displayWidth(line))
	}
	return width
}

// constrainTableWidths shrinks the widest columns until the rendered table
// fits within a positive terminal display-width limit.
func constrainTableWidths(widths []int, style string, maximum int) []int {
	constrained := slices.Clone(widths)
	if maximum <= 0 || len(constrained) == 0 {
		return constrained
	}
	overhead := 2 * (len(constrained) - 1)
	if style != "compact" {
		overhead = 3*len(constrained) + 1
	}
	available := maximum - overhead
	const minimumColumnWidth = 2
	if available < minimumColumnWidth*len(constrained) {
		return constrained
	}
	total := 0
	for _, width := range constrained {
		total += width
	}
	for total > available {
		widest := 0
		for index, width := range constrained {
			if width > constrained[widest] {
				widest = index
			}
		}
		constrained[widest]--
		total--
	}
	return constrained
}

// renderCompactTable renders whitespace-separated rows without borders.
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

// writeCompactLine appends one compact table header or data row.
func writeCompactLine(b *strings.Builder, columns []Column, row map[string]string, widths []int, header bool) {
	cells, height := wrappedTableCells(columns, row, widths, header)
	for line := range height {
		for i := range columns {
			value := wrappedCellLine(cells[i], line)
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
}

// boxStyle contains the border glyphs used by boxed table renderers.
type boxStyle struct {
	topLeft, topSeparator, topRight          string
	middleLeft, middleSeparator, middleRight string
	bottomLeft, bottomSeparator, bottomRight string
	vertical, horizontal                     string
}

// asciiBox is the plain ASCII border style.
var asciiBox = boxStyle{
	topLeft: "+", topSeparator: "+", topRight: "+",
	middleLeft: "+", middleSeparator: "+", middleRight: "+",
	bottomLeft: "+", bottomSeparator: "+", bottomRight: "+",
	vertical: "|", horizontal: "-",
}

// lightBox is the Unicode light-line border style.
var lightBox = boxStyle{
	topLeft: "┌", topSeparator: "┬", topRight: "┐",
	middleLeft: "├", middleSeparator: "┼", middleRight: "┤",
	bottomLeft: "└", bottomSeparator: "┴", bottomRight: "┘",
	vertical: "│", horizontal: "─",
}

// renderBoxTable renders rows with a border style.
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

// writeBoxLine appends one bordered header or data row.
func writeBoxLine(b *strings.Builder, columns []Column, row map[string]string, widths []int, header bool, vertical string) {
	cells, height := wrappedTableCells(columns, row, widths, header)
	for line := range height {
		b.WriteString(vertical)
		for i := range columns {
			value := wrappedCellLine(cells[i], line)
			b.WriteString(" ")
			b.WriteString(value)
			b.WriteString(strings.Repeat(" ", widths[i]-displayWidth(value)))
			b.WriteString(" ")
			b.WriteString(vertical)
		}
		b.WriteByte('\n')
	}
}

// wrappedTableCells splits one logical table row into display-width-bounded cells.
func wrappedTableCells(columns []Column, row map[string]string, widths []int, header bool) ([][]string, int) {
	cells := make([][]string, len(columns))
	height := 1
	for index, column := range columns {
		value := column.Label
		if !header {
			value = row[column.Key]
		}
		cells[index] = wrapDisplayText(value, widths[index])
		if len(cells[index]) > height {
			height = len(cells[index])
		}
	}
	return cells, height
}

// wrapDisplayText wraps prose at whitespace and common machine-token
// separators before falling back to a hard display-width boundary.
func wrapDisplayText(value string, width int) []string {
	if width <= 0 {
		return []string{value}
	}
	var lines []string
	for segment := range strings.SplitSeq(value, "\n") {
		if displayWidth(segment) <= width {
			lines = append(lines, segment)
			continue
		}
		for {
			line := wrappedPrefix(segment, width)
			lines = append(lines, line)
			if len(line) == len(segment) {
				break
			}
			segment = trimWrapSpace(segment[len(line):])
		}
	}
	return lines
}

// wrappedPrefix finds the next physical line, preferring word and separator boundaries.
// Each candidate ends at a complete grapheme. Only the current line and one lookahead
// cluster are measured, so long cells do not repeatedly scan their unrendered suffix.
func wrappedPrefix(text string, limit int) string {
	width, wordBreak, separatorBreak, lastContent := 0, 0, 0, 0
	clusters := graphemes.FromString(text)
	for clusters.Next() {
		cluster := clusters.Value()
		width += displayWidth(cluster)
		if width > limit {
			if strings.TrimSpace(cluster) == "" {
				return text[:clusters.Start()]
			}
			if wordBreak > 0 {
				return text[:wordBreak]
			}
			if separatorBreak > 0 {
				return text[:separatorBreak]
			}
			return text[:clusters.Start()]
		}
		if strings.TrimSpace(cluster) == "" {
			wordBreak = lastContent
			continue
		}
		lastContent = clusters.End()
		first, _ := utf8.DecodeRuneInString(cluster)
		if strings.ContainsRune("/\\-_", first) {
			separatorBreak = lastContent
		}
	}
	return text
}

// trimWrapSpace skips whitespace between wrapped lines without detaching combining marks.
func trimWrapSpace(text string) string {
	clusters := graphemes.FromString(text)
	for clusters.Next() {
		if strings.TrimSpace(clusters.Value()) != "" {
			return text[clusters.Start():]
		}
	}
	return ""
}

// wrappedCellLine returns one physical line or an empty continuation cell.
func wrappedCellLine(lines []string, index int) string {
	if index >= len(lines) {
		return ""
	}
	return lines[index]
}

// writeBorder appends one horizontal borderline for the supplied widths.
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

// displayWidth returns the terminal display width of value.
func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}
