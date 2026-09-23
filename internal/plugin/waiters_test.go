/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestWaiterBindingsAndDiscovery prevents unsafe polling, missing references,
// and contradictory terminal conditions from reaching command execution.
func TestWaiterBindingsAndDiscovery(t *testing.T) {
	read := Command{ID: "read", Operation: "get"}
	mutate := Command{ID: "mutate", Operation: "put"}
	dangerous := Command{ID: "dangerous", Operation: "get", Dangerous: Dangerous{Confirm: "yes"}}
	bundle := Bundle{Commands: Commands{Commands: []Command{read, mutate, dangerous}}, APIs: APIs{Operations: map[string]Operation{"get": {Retryable: true}, "put": {Retryable: false}}}}
	for _, tc := range []struct {
		name  string
		spec  Waiter
		valid bool
	}{
		{"legacy", Waiter{Path: "state", Success: "ok"}, true},
		{"bound", Waiter{Commands: []string{"read"}, Path: "state", Success: "ok", Failure: "bad"}, true},
		{"multiple", Waiter{Commands: []string{"read"}, Path: "state", SuccessValues: []string{"ready", "active"}, FailureValues: []string{"bad", "expired"}}, true},
		{"missing path", Waiter{Success: "ok"}, false},
		{"missing success", Waiter{Path: "state"}, false},
		{"overlap", Waiter{Path: "state", Success: "ok", FailureValues: []string{"ok"}}, false},
		{"empty success", Waiter{Path: "state", SuccessValues: []string{""}}, false},
		{"empty failure", Waiter{Path: "state", Success: "ok", FailureValues: []string{""}}, false},
		{"unknown command", Waiter{Commands: []string{"unknown"}, Path: "state", Success: "ok"}, false},
		{"mutation", Waiter{Commands: []string{"mutate"}, Path: "state", Success: "ok"}, false},
		{"dangerous", Waiter{Commands: []string{"dangerous"}, Path: "state", Success: "ok"}, false},
		{"negative limit", Waiter{Path: "state", Success: "ok", MaxAttempts: -1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle.Waiters = Waiters{Waiters: map[string]Waiter{"ready": tc.spec}}
			if err := ValidateWaiterBindings(bundle); (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
	bundle.Waiters = Waiters{Waiters: map[string]Waiter{"z": {Commands: []string{"read"}}, "a": {}, "other": {Commands: []string{"other"}}}}
	if got := CommandWaiters(bundle, read); !reflect.DeepEqual(got, []string{"a", "z"}) {
		t.Fatalf("read waiters %v", got)
	}
	for _, command := range []Command{mutate, dangerous} {
		if got := CommandWaiters(bundle, command); len(got) > 0 {
			t.Fatalf("unsafe waiters %v", got)
		}
	}
}

// TestLoadRejectsInvalidWaiterConditions checks validation through bundle loading.
func TestLoadRejectsInvalidWaiterConditions(t *testing.T) {
	dir := writeBundle(t, "ecs", ">=0.4.0 <1.0.0")
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{"waiters":{"invalid":{"path":"state","success":"same","failure":"same"}}}`)
	if _, err := LoadBundle(dir, "0.4.0"); err == nil {
		t.Fatal("loaded contradictory waiter")
	}
}

// TestCollectionWaiterRequiresResolvableIdentity prevents selectors from
// silently referring to an absent CLI option or an unbound command.
func TestCollectionWaiterRequiresResolvableIdentity(t *testing.T) {
	command := Command{ID: "show", Path: []string{"test", "show", "{id}"}, Parameters: []Parameter{{Name: "resource_id", Flag: "resource-id"}}}
	for _, tc := range []struct{ source, want string }{{"$arg.id", "{id}"}, {"$param.resource_id", "--resource-id"}, {"$arg.other", ""}, {"$param.other", ""}, {"literal", ""}} {
		spec := Waiter{Commands: []string{"show"}, Path: "state", Success: "ready", Selector: &waiter.Selector{Path: "items", Key: "id", Value: tc.source}}
		if got := WaiterSelectorInput(command, spec); got != tc.want {
			t.Fatalf("%s: %s", tc.source, got)
		}
		bundle := Bundle{Commands: Commands{Commands: []Command{command}}, Waiters: Waiters{Waiters: map[string]Waiter{"ready": spec}}}
		if err := ValidateWaiterBindings(bundle); (err == nil) != (tc.want != "") {
			t.Fatalf("%s validation: %v", tc.source, err)
		}
		spec.Selector.Path = ""
		bundle.Waiters.Waiters["ready"] = spec
		if err := ValidateWaiterBindings(bundle); err == nil {
			t.Fatal("empty selector path accepted")
		}
	}
	if got := WaiterSelectorInput(command, Waiter{}); got != "" {
		t.Fatal(got)
	}
}

// TestXMLWaiterBindings requires an explicit XML-only retrieval and a single selection model.
func TestXMLWaiterBindings(t *testing.T) {
	spec := Waiter{Commands: []string{"show"}, XMLPath: []apicontract.XMLName{{Local: "Object"}, {Local: "State"}}, Success: "ready"}
	command := Command{ID: "show", Operation: "show"}
	operation := Operation{Method: "GET", Retryable: true, Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "xml"}}}}
	bundle := Bundle{Commands: Commands{Commands: []Command{command}}, APIs: APIs{Operations: map[string]Operation{"show": operation}}, Waiters: Waiters{Waiters: map[string]Waiter{"ready": spec}}}
	if err := ValidateWaiterBindings(bundle); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Waiter){func(w *Waiter) { w.Path = "state" }, func(w *Waiter) { w.Selector = &waiter.Selector{Path: "rows", Key: "id", Value: "$arg.id"} }, func(w *Waiter) { w.Commands = nil }, func(w *Waiter) { w.XMLPath[0].Local = "bad/name" }} {
		bad := spec
		bad.XMLPath = append([]apicontract.XMLName{}, spec.XMLPath...)
		change(&bad)
		bundle.Waiters.Waiters["ready"] = bad
		if err := ValidateWaiterBindings(bundle); err == nil {
			t.Fatal("ambiguous XML waiter accepted", bad)
		}
	}
	operation.Response = nil
	bundle.APIs.Operations["show"] = operation
	if WaiterApplies(bundle, command, spec) {
		t.Fatal("legacy operation accepted XML waiter")
	}
}
