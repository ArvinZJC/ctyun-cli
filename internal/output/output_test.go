package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderTableUsesSelectedColumnsAndLabels(t *testing.T) {
	rows := []map[string]string{
		{"instance_id": "ins-1", "name": "web", "status": "running", "private_ip": "10.0.0.2"},
	}
	columns := []Column{
		{Key: "instance_id", Label: "Instance ID"},
		{Key: "name", Label: "Name"},
		{Key: "status", Label: "Status"},
		{Key: "private_ip", Label: "Private IP"},
	}

	got, err := RenderTable(rows, columns, TableOptions{Columns: []string{"instance_id", "status"}})
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	if !strings.Contains(got, "Instance ID") || !strings.Contains(got, "Status") {
		t.Fatalf("rendered table is missing selected headers:\n%s", got)
	}
	if strings.Contains(got, "Private IP") {
		t.Fatalf("rendered table contains unselected header:\n%s", got)
	}
}

func TestRenderTableCanHideHeader(t *testing.T) {
	got, err := RenderTable(
		[]map[string]string{{"instance_id": "ins-1"}},
		[]Column{{Key: "instance_id", Label: "Instance ID"}},
		TableOptions{NoHeader: true},
	)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}
	if strings.Contains(got, "Instance ID") {
		t.Fatalf("rendered table contains header despite NoHeader:\n%s", got)
	}
}

func TestRenderTableDefaultsToBorderedStyle(t *testing.T) {
	got, err := RenderTable(
		[]map[string]string{{"name": "华东1", "status": "running"}},
		[]Column{{Key: "name", Label: "资源池名称"}, {Key: "status", Label: "Status"}},
		TableOptions{},
	)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}
	for _, want := range []string{"┌", "┬", "│ 资源池名称 ", "├", "└"} {
		if !strings.Contains(got, want) {
			t.Fatalf("default bordered table missing %q:\n%s", want, got)
		}
	}
}

func TestRenderTableAlignsWideCharacters(t *testing.T) {
	got, err := RenderTable(
		[]map[string]string{
			{"name": "华东1", "status": "running"},
			{"name": "prod", "status": "stopped"},
		},
		[]Column{{Key: "name", Label: "资源池名称"}, {Key: "status", Label: "Status"}},
		TableOptions{Style: "compact"},
	)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	for _, want := range []string{
		"华东1       running",
		"prod        stopped",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wide character alignment missing %q:\n%s", want, got)
		}
	}
}

func TestRenderTableAlignsEmojiWidth(t *testing.T) {
	got, err := RenderTable(
		[]map[string]string{
			{"name": "✅✅", "status": "running"},
			{"name": "prod", "status": "stopped"},
		},
		[]Column{{Key: "name", Label: "名称"}, {Key: "status", Label: "Status"}},
		TableOptions{Style: "compact"},
	)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}

	if !strings.Contains(got, "✅✅  running") {
		t.Fatalf("emoji-width alignment is off:\n%s", got)
	}
}

func TestRenderTableSupportsBorderedStyle(t *testing.T) {
	got, err := RenderTable(
		[]map[string]string{{"name": "华东1", "status": "running"}},
		[]Column{{Key: "name", Label: "资源池名称"}, {Key: "status", Label: "Status"}},
		TableOptions{Style: "bordered"},
	)
	if err != nil {
		t.Fatalf("RenderTable returned error: %v", err)
	}
	for _, want := range []string{"┌", "┬", "│ 资源池名称 ", "├", "└"} {
		if !strings.Contains(got, want) {
			t.Fatalf("bordered table missing %q:\n%s", want, got)
		}
	}
}

func TestRenderJSONPreservesOriginalPayload(t *testing.T) {
	payload := map[string]any{
		"returnObj": map[string]any{
			"instances": []any{map[string]any{"instanceID": "ins-1"}},
		},
	}

	got, err := RenderJSON(payload)
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("RenderJSON output is not JSON: %v", err)
	}
	if _, ok := decoded["returnObj"]; !ok {
		t.Fatalf("RenderJSON did not preserve returnObj: %s", got)
	}
}

func TestFilterRowsUsesStableKeys(t *testing.T) {
	rows := []map[string]string{
		{"instance_id": "ins-1", "status": "running"},
		{"instance_id": "ins-2", "status": "stopped"},
	}

	filtered, err := FilterRows(rows, "status=running")
	if err != nil {
		t.Fatalf("FilterRows returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0]["instance_id"] != "ins-1" {
		t.Fatalf("filtered rows = %#v, want only ins-1", filtered)
	}

	filtered, err = FilterRows(rows, "status!=running")
	if err != nil {
		t.Fatalf("FilterRows returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0]["instance_id"] != "ins-2" {
		t.Fatalf("filtered rows = %#v, want only ins-2", filtered)
	}
}

func TestSortRowsUsesStableKeys(t *testing.T) {
	rows := []map[string]string{
		{"instance_id": "ins-2", "name": "worker"},
		{"instance_id": "ins-1", "name": "api"},
	}

	sorted, err := SortRows(rows, "instance_id")
	if err != nil {
		t.Fatalf("SortRows returned error: %v", err)
	}
	if sorted[0]["instance_id"] != "ins-1" {
		t.Fatalf("ascending sort = %#v", sorted)
	}

	sorted, err = SortRows(rows, "-instance_id")
	if err != nil {
		t.Fatalf("SortRows returned error: %v", err)
	}
	if sorted[0]["instance_id"] != "ins-2" {
		t.Fatalf("descending sort = %#v", sorted)
	}
}
