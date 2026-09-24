/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestZOSMigrationRateControls checks the reviewed rate-limit request contract.
func TestZOSMigrationRateControls(t *testing.T) {
	ctx := loadStorageReviewContext(t, "zos")
	const id = "v4.zos.migration.create"
	for _, tc := range []struct {
		field, name string
		kind        plugin.ParameterValueType
	}{
		{"rateLimitType", "rate_limit_type", ""},
		{"rateLimitNum", "rate_limit_num", plugin.ParameterValueInteger},
		{"rateLimitPolicy", "rate_limit_policy", plugin.ParameterValueJSON},
		{"consistencyCheck", "consistency_check", plugin.ParameterValueBoolean},
	} {
		p := ctx.commandParameter(id, tc.field)
		if p.ValueType != tc.kind || ctx.bundle.APIs.Operations[id].Body[tc.field] != "$param."+tc.name {
			t.Errorf("%s: type=%s body=%v", tc.field, p.ValueType, ctx.bundle.APIs.Operations[id].Body)
		}
	}
}

// TestZOSMigrationRateControlsCLI exercises help, completion, and local validation.
func TestZOSMigrationRateControlsCLI(t *testing.T) {
	for _, lang := range []string{"en-US", "en-GB", "zh-CN"} {
		var out bytes.Buffer
		if err := cli.Run(cli.Config{Args: []string{"--lang", lang, "zos", "migration", "create", "--help"}, Stdout: &out, PluginRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"--rate-limit-type", "--rate-limit-num", "--rate-limit-policy", "--consistency-check", "CRC64", "Regex", "File_list", "destinationPrefix"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s help missing %s", lang, want)
			}
		}
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"zos", "migration", "create", ""}, "--rate-limit-policy"},
		{[]string{"zos", "migration", "create", "--rate-l"}, "--rate-limit-policy"},
		{[]string{"zos", "migration", "create", "--rate-limit-type", ""}, "Period"},
	} {
		var out bytes.Buffer
		if err := cli.Run(cli.Config{Args: append([]string{"__complete"}, tc.args...), Stdout: &out, PluginRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("completion %v missing %s: %s", tc.args, tc.want, out.String())
		}
	}
	base := []string{"--lang", "en-US", "zos", "migration", "create", "--migration-name", "aanjyy", "--region", "81f7728xxxxx0155d307d5b", "--migration-mode", "semi-managed", "--migration-agent", "111_agt_76f7dc8b012xxxxf6ed082db55d08", "--source-info", "[]", "--destination-info", "[]", "--yes", "--offline"}
	for _, tc := range []struct {
		name  string
		extra []string
		want  string
	}{
		{"global requires limit", []string{"--rate-limit-type", "Global"}, "--rate-limit-num"},
		{"period requires policy", []string{"--rate-limit-type", "Period"}, "--rate-limit-policy"},
		{"minimum", []string{"--rate-limit-type", "Global", "--rate-limit-num", "49"}, "--rate-limit-num"},
		{"maximum", []string{"--rate-limit-type", "Global", "--rate-limit-num", "500001"}, "--rate-limit-num"},
		{"global valid", []string{"--rate-limit-type", "Global", "--rate-limit-num", "200", "--consistency-check", "true"}, ""},
		{"period valid", []string{"--rate-limit-type", "Period", "--rate-limit-policy", `{"else":-1,"08:20-08:40":3000,"18:20-18:40":4000}`}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cli.Run(cli.Config{Args: append(append([]string{}, base...), tc.extra...), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, PluginRoot: t.TempDir()})
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %s", err, tc.want)
			}
		})
	}
}

// TestZOSMigrationRefreshedFixtures preserves newly documented nested output.
func TestZOSMigrationRefreshedFixtures(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"migration", "show", "--migration-id", "111_mig_76f7dc8b012xxxxf6ed082db55d08"}, []string{`"rateLimitType": "Global"`, `"rateLimitNum": 200`, `"consistencyCheck": true`, `"migrateRegex": []`, `"migrateFileList": []`}},
		{[]string{"migration", "show-history", "--migration-id", "222_mig_f4416xxxxx5a2695318f"}, []string{`"errorColumn": null`}},
	} {
		var out bytes.Buffer
		args := append([]string{"--lang", "en-US", "zos"}, tc.args...)
		args = append(args, "--region", "81f7728xxxxx0155d307d5b", "--offline", "--output", "json")
		if err := cli.Run(cli.Config{Args: args, Stdout: &out, PluginRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out.String(), want) {
				t.Errorf("output missing %s: %s", want, out.String())
			}
		}
	}
}

// TestZOSMigrationRateWireTypes prevents JSON policy and Boolean inputs from becoming strings.
func TestZOSMigrationRateWireTypes(t *testing.T) {
	for _, tc := range []struct {
		mode    string
		options []string
		field   string
		want    any
	}{
		{"Global", []string{"--rate-limit-num", "200"}, "rateLimitNum", float64(200)},
		{"Period", []string{"--rate-limit-policy", `{"else":-1,"08:20-08:40":3000,"18:20-18:40":4000}`}, "rateLimitPolicy", map[string]any{"else": float64(-1), "08:20-08:40": float64(3000), "18:20-18:40": float64(4000)}},
	} {
		transport := &refreshTransport{}
		args := []string{"zos", "migration", "create", "--migration-name", "aanjyy", "--region", "81f7728xxxxx0155d307d5b", "--migration-mode", "semi-managed", "--migration-agent", "111_agt_76f7dc8b012xxxxf6ed082db55d08", "--source-info", "[]", "--destination-info", "[]", "--yes", "--output", "json", "--consistency-check", "true", "--rate-limit-type", tc.mode}
		args = append(args, tc.options...)
		err := cli.Run(cli.Config{Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(key string) string {
			if key == "CTYUN_AK" || key == "CTYUN_SK" {
				return "test-credential"
			}
			return ""
		}})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(transport.body[tc.field], tc.want) || transport.body["consistencyCheck"] != true || transport.body["rateLimitType"] != tc.mode {
			t.Fatalf("typed body=%v", transport.body)
		}
	}
}

// TestZOSMigrationRateTableColumns checks both locale labels and stable selectors.
func TestZOSMigrationRateTableColumns(t *testing.T) {
	for _, tc := range []struct{ language, columns, want string }{
		{"en-US", "rate_limit_type,rate_limit_num,consistency_check", "Global"},
		{"zh-CN", "限流方式,全时段限速值,数据一致性校验", "Global"},
	} {
		var out bytes.Buffer
		err := cli.Run(cli.Config{Args: []string{"--lang", tc.language, "zos", "migration", "show", "--migration-id", "111_mig_76f7dc8b012xxxxf6ed082db55d08", "--region", "81f7728xxxxx0155d307d5b", "--cols", tc.columns, "--offline"}, Stdout: &out, PluginRoot: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) || !strings.Contains(out.String(), "200") || !strings.Contains(out.String(), "true") {
			t.Fatalf("%s selected table: %s", tc.language, out.String())
		}
	}
}
