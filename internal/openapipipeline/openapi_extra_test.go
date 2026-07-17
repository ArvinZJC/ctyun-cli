/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestValidAcceptedStatusGuardPathAcceptsResultData(t *testing.T) {
	if !validAcceptedStatusGuardPath("returnObj.satisfied") {
		t.Fatal("validAcceptedStatusGuardPath rejected result data")
	}
}

func TestCatalogValidationRejectsAnnotationShapes(t *testing.T) {
	cases := []catalogValidationCase{
		{name: "invalid accepted status", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "ok"}}
		}, want: "operation v4.ecs.instance.list accepted status code ok is invalid"},
		{name: "unsupported accepted status", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "901", RequiredPath: "returnObj.satisfied"}}
		}, want: "operation v4.ecs.instance.list accepted status code 901 is invalid"},
		{name: "missing accepted path", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "900"}}
		}, want: "operation v4.ecs.instance.list accepted status path  is invalid"},
		{name: "invalid accepted path", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "900", RequiredPath: "returnObj..satisfied"}}
		}, want: "operation v4.ecs.instance.list accepted status path returnObj..satisfied is invalid"},
		{name: "error envelope accepted path", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "900", RequiredPath: "errorCode"}}
		}, want: "operation v4.ecs.instance.list accepted status path errorCode is invalid"},
		{name: "unknown conditional parameter", mutate: func(catalog *Catalog) {
			catalog.Operations[0].ConditionalRequirements = []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "missing", Equals: "ecs"},
				Required: []string{"name"},
			}}
		}, want: "operation v4.ecs.instance.list conditional parameter missing is unknown"},
		{name: "missing conditional match", mutate: func(catalog *Catalog) {
			catalog.Operations[0].ConditionalRequirements = []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "name"},
				Required: []string{"name"},
			}}
		}, want: "operation v4.ecs.instance.list conditional parameter name has no match value"},
		{name: "missing conditional requirements", mutate: func(catalog *Catalog) {
			catalog.Operations[0].ConditionalRequirements = []plugin.ConditionalRequirement{{
				When: plugin.ParameterCondition{Parameter: "name", Equals: "ecs"},
			}}
		}, want: "operation v4.ecs.instance.list conditional parameter name has no requirements"},
		{name: "unknown conditional requirement", mutate: func(catalog *Catalog) {
			catalog.Operations[0].ConditionalRequirements = []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "name", Equals: "ecs"},
				Required: []string{"missing"},
			}}
		}, want: "operation v4.ecs.instance.list conditional requirement missing is unknown"},
	}
	runCatalogValidationCases(t, cases)
}

func TestAcceptedStatusAndResponsePathValidation(t *testing.T) {
	if validStatusCode("") {
		t.Fatal("validStatusCode accepted empty status")
	}
	if !validStatusCode("900") {
		t.Fatal("validStatusCode rejected numeric status")
	}
	if validResponsePath("returnObj..satisfied") {
		t.Fatal("validResponsePath accepted empty segment")
	}
	for _, path := range []string{".returnObj", "returnObj.", "returnObj.9bad", "returnObj.bad-name"} {
		if validResponsePath(path) {
			t.Fatalf("validResponsePath accepted invalid path %q", path)
		}
	}
	if !validResponsePath("returnObj.satisfied") {
		t.Fatal("validResponsePath rejected dotted object path")
	}
	if !validResponsePath("return_obj.satisfied_1") {
		t.Fatal("validResponsePath rejected underscores and digits")
	}
}

func TestCommandActionHeuristics(t *testing.T) {
	cases := []struct {
		operation Operation
		want      string
	}{
		{operation: Operation{Path: "/v4/demo/list-items"}, want: "list"},
		{operation: Operation{Path: "/v4/demo/item-list"}, want: "list"},
		{operation: Operation{Path: "/v4/demo/details"}, want: "show"},
		{operation: Operation{ID: "v4.demo.delete"}, want: "delete"},
		{operation: Operation{}, want: "call"},
	}
	for _, tc := range cases {
		if got := commandAction(tc.operation); got != tc.want {
			t.Fatalf("commandAction(%#v) = %q, want %q", tc.operation, got, tc.want)
		}
	}
}
