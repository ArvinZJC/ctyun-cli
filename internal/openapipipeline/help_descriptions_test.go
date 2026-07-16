/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParameterHelpDescriptionsJSONRoundTrip(t *testing.T) {
	var parameter Parameter
	if err := json.Unmarshal([]byte(`{"name":"resourcePoolID","help_descriptions":{"zh-CN":"4.0 资源池必填；单区填写 default"}}`), &parameter); err != nil {
		t.Fatalf("unmarshal parameter: %v", err)
	}
	data, err := json.Marshal(parameter)
	if err != nil {
		t.Fatalf("marshal parameter: %v", err)
	}
	if !strings.Contains(string(data), `"help_descriptions":{"zh-CN":"4.0 资源池必填；单区填写 default"}`) {
		t.Fatalf("marshaled parameter = %s, want reviewed help_descriptions", data)
	}
}

func TestParameterEnglishDescriptionPrefersReviewedBritishEnglish(t *testing.T) {
	var parameter Parameter
	if err := json.Unmarshal([]byte(`{
		"name":"version",
		"description":"source English",
		"descriptions":{"en-GB":"source British English","en-US":"source American English"},
		"help_descriptions":{"en-GB":"Reviewed British English","en-US":"Reviewed American English"}
	}`), &parameter); err != nil {
		t.Fatalf("unmarshal parameter: %v", err)
	}
	if got := parameterEnglishDescription(parameter); got != "Reviewed British English" {
		t.Fatalf("English parameter description = %q, want reviewed en-GB text", got)
	}
}

func TestParameterEnglishDescriptionFallsBackToReviewedAmericanEnglish(t *testing.T) {
	var parameter Parameter
	if err := json.Unmarshal([]byte(`{
		"name":"version",
		"description":"source English",
		"descriptions":{"en-GB":"source British English"},
		"help_descriptions":{"en-US":"Reviewed American English"}
	}`), &parameter); err != nil {
		t.Fatalf("unmarshal parameter: %v", err)
	}
	if got := parameterEnglishDescription(parameter); got != "Reviewed American English" {
		t.Fatalf("English parameter description = %q, want reviewed en-US text", got)
	}
}

func TestParameterLocalizedDescriptionPreservesReviewedTextExactly(t *testing.T) {
	descriptions := []string{
		"4.0 资源池必填；单区填写 default",
		"弹性 IP 版本；默认 ipv4",
		"安全防护类型；默认 false",
		"自动续订状态：0 关闭，1 开启",
	}
	for _, description := range descriptions {
		t.Run(description, func(t *testing.T) {
			var parameter Parameter
			encoded, err := json.Marshal(map[string]any{
				"name":              "reviewedValue",
				"descriptions":      map[string]string{"zh-CN": "原始说明"},
				"help_descriptions": map[string]string{"zh-CN": description},
			})
			if err != nil {
				t.Fatalf("marshal parameter fixture: %v", err)
			}
			if err := json.Unmarshal(encoded, &parameter); err != nil {
				t.Fatalf("unmarshal parameter: %v", err)
			}
			if got := parameterLocalizedDescription(parameter, "zh-CN"); got != description {
				t.Fatalf("localized parameter description = %q, want exact reviewed text %q", got, description)
			}
		})
	}
}

func TestOperationValidateAcceptsPartialReviewedHelpDescriptions(t *testing.T) {
	catalog := loadCatalogFixture(t)
	var parameter Parameter
	if err := json.Unmarshal([]byte(`{
		"name":"regionID",
		"location":"body",
		"type":"string",
		"help_descriptions":{"zh-CN":"资源池 ID"}
	}`), &parameter); err != nil {
		t.Fatalf("unmarshal parameter: %v", err)
	}
	catalog.Operations[0].Parameters[0] = parameter
	if err := catalog.Validate(); err != nil {
		t.Fatalf("Catalog.Validate rejected partial reviewed help descriptions: %v", err)
	}
}

func TestOperationValidateRejectsInvalidReviewedHelpDescriptions(t *testing.T) {
	tests := []struct {
		name             string
		helpDescriptions map[string]string
		want             string
	}{
		{
			name:             "unsupported language",
			helpDescriptions: map[string]string{"fr-FR": "Description"},
			want:             "help_descriptions language fr-FR is unsupported",
		},
		{
			name:             "empty reviewed text",
			helpDescriptions: map[string]string{"en-GB": " \t"},
			want:             "help_descriptions en-GB must not be empty",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			catalog := loadCatalogFixture(t)
			encoded, err := json.Marshal(map[string]any{
				"name":              "regionID",
				"location":          "body",
				"type":              "string",
				"help_descriptions": tc.helpDescriptions,
			})
			if err != nil {
				t.Fatalf("marshal parameter fixture: %v", err)
			}
			if err := json.Unmarshal(encoded, &catalog.Operations[0].Parameters[0]); err != nil {
				t.Fatalf("unmarshal parameter: %v", err)
			}
			err = catalog.Validate()
			if err == nil {
				t.Fatalf("Catalog.Validate accepted help_descriptions %#v", tc.helpDescriptions)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Catalog.Validate error = %q, want substring %q", err, tc.want)
			}
		})
	}
}
