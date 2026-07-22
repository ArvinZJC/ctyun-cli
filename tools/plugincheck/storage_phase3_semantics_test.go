/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestZOSReviewedMetadataPreservesPortalSemantics prevents reviewed ZOS
// release, request, safety, fixture, output, localization, and lifecycle
// semantics from regressing.
func TestZOSReviewedMetadataPreservesPortalSemantics(t *testing.T) {
	context := loadStorageReviewContext(t, "zos")
	catalog, bundle := context.catalog, context.bundle
	operations, commands := context.operations, context.commands
	findSourceParameter := context.sourceParameter
	findCommandParameter := context.commandParameter

	t.Run("release shape and safety", func(t *testing.T) {
		if len(catalog.Operations) != 106 || len(bundle.APIs.Operations) != 106 || len(bundle.Commands.Commands) != 106 || len(bundle.Tables.Tables) != 106 {
			t.Fatalf("catalog/APIs/commands/tables = %d/%d/%d/%d, want 106 each", len(catalog.Operations), len(bundle.APIs.Operations), len(bundle.Commands.Commands), len(bundle.Tables.Tables))
		}
		readOnlyPOST := map[string]bool{"6281": true, "6301": true, "9845": true}
		getCount, postCount, safeCount, dangerousCount := 0, 0, 0, 0
		for _, operation := range catalog.Operations {
			if operation.Method == "GET" {
				getCount++
			} else if operation.Method == "POST" {
				postCount++
			} else {
				t.Errorf("%s has unsupported method %s", operation.ID, operation.Method)
			}
			wantSafe := operation.Method == "GET" || readOnlyPOST[operation.APIID]
			command := commands[operation.ID]
			api := bundle.APIs.Operations[operation.ID]
			if operation.Retryable != wantSafe || operation.Dangerous == wantSafe || api.Retryable != wantSafe || (command.Dangerous.Confirm == "") == !wantSafe {
				t.Errorf("%s safety differs: source retry/danger=%t/%t API retry=%t confirm=%q", operation.ID, operation.Retryable, operation.Dangerous, api.Retryable, command.Dangerous.Confirm)
			}
			if wantSafe {
				safeCount++
			} else {
				dangerousCount++
			}
		}
		if getCount != 45 || postCount != 61 || safeCount != 48 || dangerousCount != 58 {
			t.Errorf("GET/POST/safe/dangerous = %d/%d/%d/%d, want 45/61/48/58", getCount, postCount, safeCount, dangerousCount)
		}
	})

	t.Run("dangerous command prompts before offline execution", func(t *testing.T) {
		var stderr bytes.Buffer
		err := cli.Run(cli.Config{
			Args:       []string{"--lang", "en-US", "zos", "service", "enable", "--region", "public", "--client-token", "847b561a-a801-4e26-ae70-e4a1fd58cfd4", "--offline"},
			Stdin:      strings.NewReader("n\n"),
			Stdout:     &bytes.Buffer{},
			Stderr:     &stderr,
			PluginRoot: t.TempDir(),
		})
		if err == nil {
			t.Fatal("declined dangerous ZOS command returned nil error")
		}
		if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
			t.Errorf("dangerous ZOS command did not prompt: %q", stderr.String())
		}
	})

	t.Run("typed finite values defaults and patterns", func(t *testing.T) {
		defaultCount, enumCount, patternCount, booleanCount := 0, 0, 0, 0
		for _, operation := range catalog.Operations {
			for _, source := range operation.Parameters {
				parameter := findCommandParameter(operation.ID, source.Name)
				if source.Default != "" {
					defaultCount++
					if parameter.Default != source.Default {
						t.Errorf("%s.%s default = %q, want %q", operation.ID, source.Name, parameter.Default, source.Default)
					}
				}
				if len(source.Enum) > 0 {
					enumCount++
					if !reflect.DeepEqual(parameter.AllowedValues, source.Enum) {
						t.Errorf("%s.%s values = %#v, want %#v", operation.ID, source.Name, parameter.AllowedValues, source.Enum)
					}
				}
				if source.Pattern != "" {
					patternCount++
					if parameter.Pattern != source.Pattern {
						t.Errorf("%s.%s pattern = %q, want %q", operation.ID, source.Name, parameter.Pattern, source.Pattern)
					}
				}
				if source.Type == "Boolean" {
					booleanCount++
					want := []string{"true", "false"}
					if !reflect.DeepEqual(source.Enum, want) || !reflect.DeepEqual(parameter.AllowedValues, want) {
						t.Errorf("%s.%s Boolean values are not true/false", operation.ID, source.Name)
					}
				}
			}
		}
		if defaultCount != 43 || enumCount != 29 || patternCount != 9 || booleanCount != 7 {
			t.Errorf("defaults/enums/patterns/Booleans = %d/%d/%d/%d, want 43/29/9/7", defaultCount, enumCount, patternCount, booleanCount)
		}
		spotChecks := map[string][]string{
			"v4.zos.bucket.create.storageType":      {"STANDARD", "STANDARD_IA", "GLACIER"},
			"v4.zos.bucket.create.AZPolicy":         {"single-az", "multi-az"},
			"v4.zos.object-acl.set.ACL":             {"private", "public-read", "authenticated-read"},
			"v4.zos.resource-package.quote.pkgType": {"zosSize", "zosMzSize", "zosBytesSend", "zosRequest", "zosRetrievalFlow", "zosRetrievalFrequency"},
			"v4.zos.migration.create.migrationMode": {"fully-managed", "semi-managed"},
			"v4.zos.migration.list.migrationStatus": {"creating", "created", "create_failed", "starting", "waiting", "executing", "stopping", "stopped", "restoring", "agent_lost", "finished", "failed", "finished_error", "retrying", "deleting"},
		}
		for key, want := range spotChecks {
			operationID, target := storageParameterKey(t, key)
			if got := findSourceParameter(operationID, target).Enum; !reflect.DeepEqual(got, want) {
				t.Errorf("%s enum = %#v, want %#v", key, got, want)
			}
		}
	})

	t.Run("localized help and conditions", func(t *testing.T) {
		for _, operation := range catalog.Operations {
			command := commands[operation.ID]
			for _, source := range operation.Parameters {
				parameter := findCommandParameter(operation.ID, source.Name)
				key := "parameter." + command.ID + "." + parameter.Name + ".description"
				for _, language := range []string{"en-GB", "en-US", "zh-CN"} {
					help := strings.TrimSpace(source.HelpDescriptions[language])
					if help == "" || strings.Contains(help, "<") || bundle.I18N[language][key] != help {
						t.Errorf("%s %s help is not complete localized source text", key, language)
					}
				}
				if source.HelpDescriptions["en-GB"] == source.Descriptions["en-GB"] || source.HelpDescriptions["en-US"] == source.Descriptions["en-US"] {
					t.Errorf("%s retains mechanical English help", key)
				}
				if !strings.ContainsFunc(source.HelpDescriptions["zh-CN"], func(char rune) bool { return unicode.Is(unicode.Han, char) }) {
					t.Errorf("%s zh-CN help is not localized", key)
				}
			}
		}
		want := []plugin.ConditionalRequirement{{When: plugin.ParameterCondition{Parameter: "migration_mode", Equals: "semi-managed"}, Required: []string{"migration_agent"}}}
		migration := operations["v4.zos.migration.create"]
		if !reflect.DeepEqual(migration.ConditionalRequirements, want) || !reflect.DeepEqual(commands[migration.ID].ConditionalRequirements, want) {
			t.Errorf("migration conditionals differ: source=%#v command=%#v", migration.ConditionalRequirements, commands[migration.ID].ConditionalRequirements)
		}
	})

	t.Run("constraints and recommendation nuance", func(t *testing.T) {
		checks := map[string][2]string{
			"v4.zos.multipart-upload.complete": {"at least 5 MB", "至少为 5 MB"},
			"v4.zos.bucket.delete":             {"neither objects nor fragments", "存在对象或存在因分段上传产生的碎片"},
			"v4.zos.bucket-versioning.set":     {"Object locking must be disabled", "没有开启对象锁定"},
			"v4.zos.object-retention.set":      {"Object locking must have been enabled", "启用对象锁定"},
			"v4.zos.bucket-replication.set":    {"one target region", "多个不同区域"},
			"v4.zos.assessment.create":         {"East China 1", "华东1"},
			"v4.zos.migration.create":          {"five fully managed", "5个全托管"},
			"v4.zos.migration.retry":           {"available worker", "可用”状态的worker"},
			"v4.zos.role.delete":               {"Every policy must be unbound", "解绑该角色下的所有策略"},
		}
		for operationID, want := range checks {
			description := operations[operationID].Description
			if !strings.Contains(description["en-GB"], want[0]) || !strings.Contains(description["en-US"], want[0]) || !strings.Contains(description["zh-CN"], want[1]) {
				t.Errorf("%s does not preserve constraint %q/%q", operationID, want[0], want[1])
			}
		}
		for _, operationID := range []string{"v4.zos.bucket-replication.set", "v4.zos.bucket-replication.list", "v4.zos.bucket-replication.list-destination-regions", "v4.zos.bucket-replication.show-progress", "v4.zos.bucket-replication.delete"} {
			if !strings.Contains(operations[operationID].Description["zh-CN"], "上海32<->杭州2") {
				t.Errorf("%s lost literal region-pair text", operationID)
			}
		}
		statistics := operations["v4.zos.bucket-statistics.show"]
		if !strings.Contains(statistics.Description["zh-CN"], "最近 30 天") || !strings.Contains(statistics.Description["zh-CN"], "建议在当前时间 30 分钟后") || statistics.Recommendation != nil {
			t.Errorf("statistics timing guidance was lost or misclassified")
		}
	})

	t.Run("captured meaningful examples", func(t *testing.T) {
		for _, command := range bundle.Commands.Commands {
			if len(command.Examples) == 0 {
				if commandNeedsPublishedExample(command) {
					t.Errorf("%s has no source-backed example for required command options", command.ID)
				}
				continue
			}
			for _, example := range command.Examples {
				if err := plugin.ValidateCommandExample(command, example); err != nil {
					t.Errorf("%s example %q: %v", command.ID, example, err)
				}
			}
		}
		meaningful := map[string][]string{
			"v4.zos.bucket-replication.set": {"--destination-bucket", "--destination-region-id", "--prefixes", "--plot", "--history"},
			"v4.zos.bucket-quota.set":       {"--enabled", "--max-size-kb", "--max-objects"},
			"v4.zos.migration.create":       {"--migration-mode semi-managed", "--migration-agent", "--source-info", "--destination-info"},
		}
		for operationID, flags := range meaningful {
			example := strings.Join(commands[operationID].Examples, "\n")
			for _, flag := range flags {
				if !strings.Contains(example, flag) {
					t.Errorf("%s example lacks %s: %q", operationID, flag, example)
				}
			}
		}
		for _, operationID := range []string{"v4.zos.bucket-acl.set", "v4.zos.object-acl.set"} {
			example := strings.Join(commands[operationID].Examples, "\n")
			if !strings.Contains(example, "--access-control-policy") || strings.Contains(example, "--acl") {
				t.Errorf("%s does not preserve the captured ACL alternative: %q", operationID, example)
			}
			for _, target := range []string{"ACL", "accessControlPolicy"} {
				help := findSourceParameter(operationID, target).HelpDescriptions
				if !strings.Contains(help["en-US"], "do not supply") || !strings.Contains(help["zh-CN"], "不能") {
					t.Errorf("%s.%s lacks mutual-exclusion help", operationID, target)
				}
			}
		}
	})

	t.Run("minimal JSON repairs and paths", func(t *testing.T) {
		responseChecks := map[string]struct {
			path string
			want any
		}{
			"v4.zos.bucket.show":    {"returnObj.zonegroup", "34c6f6cb-fd52-4676-b7d2-0e799a644bb7"},
			"v4.zos.bucket.list":    {"returnObj.bucketList.2.bucket", "bucket-2ac4"},
			"v4.zos.region.list":    {"returnObj.2.regionName", "上海1"},
			"v4.zos.migration.list": {"returnObj.results.0.lastOperationErrorCode", float64(21105)},
		}
		for operationID, check := range responseChecks {
			var value any
			if err := json.Unmarshal(operations[operationID].ExampleResponse, &value); err != nil {
				t.Errorf("%s repaired response is invalid JSON: %v", operationID, err)
				continue
			}
			for part := range strings.SplitSeq(check.path, ".") {
				if object, ok := value.(map[string]any); ok {
					value = object[part]
					continue
				}
				if list, ok := value.([]any); ok {
					index := int(part[0] - '0')
					value = list[index]
				}
			}
			if !reflect.DeepEqual(value, check.want) {
				t.Errorf("%s %s = %#v, want %#v", operationID, check.path, value, check.want)
			}
		}
		requestChecks := map[string]map[string]string{
			"v4.zos.assessment.start":  {"evaluationID": "111_eva_c9axxxxx263d4518afb", "regionID": "81f7728xxxxx0155d307d5b"},
			"v4.zos.assessment.pause":  {"evaluationID": "111_eva_c9axxxxx263d4518afb", "regionID": "81f7728xxxxx0155d307d5b"},
			"v4.zos.assessment.resume": {"evaluationID": "111_eva_c9axxxxx263d4518afb", "regionID": "81f7728xxxxx0155d307d5b"},
			"v4.zos.migration.retry":   {"migrationID": "222_mig_a0d60xxxxxxdbeb0213362217328", "regionID": "81f7728xxxxx0155d307d5b"},
		}
		for operationID, want := range requestChecks {
			got := make(map[string]string, len(operations[operationID].RequestExample))
			for key, raw := range operations[operationID].RequestExample {
				var value string
				if err := json.Unmarshal(raw, &value); err != nil {
					t.Errorf("%s repaired request %s: %v", operationID, key, err)
				}
				got[key] = value
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s repaired request = %#v, want %#v", operationID, got, want)
			}
		}
	})

	t.Run("official fixtures and default fields", func(t *testing.T) {
		assertStorageFixtureDefaults(t, context, storageFixtureOptions{})
		credential, err := os.ReadFile(repoPath(t, "plugins/zos/fixtures/zos-credential-show.json"))
		if err != nil {
			t.Fatal(err)
		}
		credentialText := string(credential)
		for _, want := range []string{"ak1xxxx", "sk1xxxxxxx", "ak2xxxx", "sk2xxxxxxx"} {
			if !strings.Contains(credentialText, want) {
				t.Errorf("credential fixture lost official dummy value %q", want)
			}
		}
	})

	t.Run("technical labels and lifecycle", func(t *testing.T) {
		labels := map[string]map[string]string{
			"v4.zos.bucket-cors.show":     {"cors_rules": "CORS 规则"},
			"v4.zos.bucket.show":          {"explicit_placement": "显式 Placement", "zonegroup": "Zone 组"},
			"v4.zos.access-endpoint.list": {"intranet_endpoint": "内网 Endpoint 列表", "internet_endpoint": "外网 Endpoint 列表"},
			"v4.zos.role.create":          {"arn": "角色 ARN"},
			"v4.zos.role.list":            {"role_arn": "角色 ARN"},
			"v4.zos.role.show":            {"role_arn": "角色 ARN"},
		}
		for operationID, expected := range labels {
			table := bundle.Tables.Tables[commands[operationID].Table]
			for _, column := range table.Columns {
				if want, exists := expected[column.Key]; exists && column.Labels["zh-CN"] != want {
					t.Errorf("%s.%s zh-CN label = %q, want %q", operationID, column.Key, column.Labels["zh-CN"], want)
				}
			}
		}
		wrongCasing := []string{"Cors", "Url", "Arn", "Acl", "Az Policy", "Cmk", "Uid", "Uuid", "Ip", "Sts", "Zos"}
		for _, operation := range catalog.Operations {
			command := commands[operation.ID]
			if operation.Recommendation != nil || command.Recommendation != nil || command.Deprecation != nil || bundle.APIs.Operations[operation.ID].Deprecation != nil {
				t.Errorf("%s has unsupported lifecycle metadata", operation.ID)
			}
			for _, parameter := range command.Parameters {
				if parameter.Deprecation != nil {
					t.Errorf("%s.%s has unsupported parameter lifecycle metadata", operation.ID, parameter.Name)
				}
			}
			for _, column := range bundle.Tables.Tables[command.Table].Columns {
				if column.Deprecation != nil {
					t.Errorf("%s.%s has unsupported column lifecycle metadata", operation.ID, column.Key)
				}
				for _, label := range column.Labels {
					for _, wrong := range wrongCasing {
						if strings.Contains(label, wrong) {
							t.Errorf("%s.%s label has incorrect casing %q", operation.ID, column.Key, label)
						}
					}
				}
			}
		}
	})
}
