package output

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
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
		options.Style = "compact"
	}
	if options.Style != "compact" && options.Style != "plain" && options.Style != "bordered" {
		return "", fmt.Errorf("unsupported table style %q", options.Style)
	}

	writer := prettytable.NewWriter()
	writer.SetStyle(renderStyle(options.Style))
	writer.SuppressTrailingSpaces()
	if !options.NoHeader {
		writer.AppendHeader(tableRowForHeader(selected))
	}
	for _, row := range rows {
		writer.AppendRow(tableRowForData(selected, row))
	}
	return writer.Render() + "\n", nil
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

func renderStyle(style string) prettytable.Style {
	keepLabels := func(style prettytable.Style) prettytable.Style {
		style.Format.Header = text.FormatDefault
		style.Format.Footer = text.FormatDefault
		return style
	}
	switch style {
	case "bordered":
		return keepLabels(prettytable.StyleLight)
	case "plain":
		return keepLabels(prettytable.StyleDefault)
	default:
		compact := prettytable.StyleDefault
		compact.Name = "ctyun-compact"
		compact.Options = prettytable.OptionsNoBordersAndSeparators
		compact.Box.PaddingLeft = ""
		compact.Box.PaddingRight = "  "
		return keepLabels(compact)
	}
}

func tableRowForHeader(columns []Column) prettytable.Row {
	row := make(prettytable.Row, 0, len(columns))
	for _, col := range columns {
		row = append(row, col.Label)
	}
	return row
}

func tableRowForData(columns []Column, values map[string]string) prettytable.Row {
	row := make(prettytable.Row, 0, len(columns))
	for _, col := range columns {
		row = append(row, values[col.Key])
	}
	return row
}
