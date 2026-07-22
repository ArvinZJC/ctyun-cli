/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "strings"

// NormalizeSourceLabels applies conservative shared label repairs to one
// tracked source catalog and returns the number of changed label values.
func (workspace Workspace) NormalizeSourceLabels(product string) (int, error) {
	catalog, err := workspace.ReadSource(product)
	if err != nil {
		return 0, err
	}
	changed := 0
	for operationIndex := range catalog.Operations {
		columns := catalog.Operations[operationIndex].Response.Columns
		for columnIndex := range columns {
			column := &columns[columnIndex]
			if label := strings.TrimSpace(column.LabelEN); label != "" {
				normalized := NormalizeDisplayLabel("en-US", label)
				if normalized != column.LabelEN {
					column.LabelEN = normalized
					changed++
				}
			}
			if label := strings.TrimSpace(column.LabelZH); label != "" {
				normalized := NormalizeDisplayLabel("zh-CN", label)
				if DisplayLabelQualityFinding("zh-CN", normalized) != "" {
					suggestion := NormalizeDisplayLabel("zh-CN", chineseNameForIdentifier(columnIdentifier(*column)))
					if DisplayLabelQualityFinding("zh-CN", suggestion) == "" {
						normalized = suggestion
					}
				}
				if DisplayLabelQualityFinding("zh-CN", normalized) == "" && normalized != column.LabelZH {
					column.LabelZH = normalized
					changed++
				}
			}
		}
		catalog.Operations[operationIndex].Response.Columns = columns
	}
	if changed == 0 {
		return 0, nil
	}
	if err := workspace.WriteCatalog(workspace.ProductPath(product, "source.json"), catalog); err != nil {
		return 0, err
	}
	return changed, nil
}
