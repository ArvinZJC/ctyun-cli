/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package waiter

import (
	"encoding/json"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
)

// TestXMLWaiterStates verifies namespaced scalar selection without order or first-row assumptions.
func TestXMLWaiterStates(t *testing.T) {
	spec := Spec{XMLPath: []apicontract.XMLName{{Space: "urn:status", Local: "Object"}, {Space: "urn:status", Local: "State"}}, Success: "ready", Failure: "failed"}
	for _, tc := range []struct {
		xml   string
		state State
		bad   bool
	}{
		{`<Object xmlns="urn:status"><Other/><State>ready</State></Object>`, Success, false},
		{`<Object xmlns="urn:status"><State>failed</State></Object>`, Failure, false},
		{`<Object xmlns="urn:status"><State/></Object>`, Pending, false},
		{`<Object xmlns="urn:status"><State>working</State></Object>`, Pending, false},
		{`<Object><State>ready</State></Object>`, Pending, true},
		{`<Object xmlns="urn:status"/>`, Pending, true},
		{`<Object xmlns="urn:status"><State>ready</State><State>ready</State></Object>`, Pending, true},
		{`<Object xmlns="urn:status"><State><Value>ready</Value></State></Object>`, Pending, true},
	} {
		node, err := client.DecodeXML([]byte(tc.xml))
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(node)
		var payload map[string]any
		if decodeErr := json.Unmarshal(data, &payload); decodeErr != nil {
			t.Error(decodeErr)
		}
		state, err := Evaluate(spec, payload)
		if (err != nil) != tc.bad || state != tc.state {
			t.Fatal(tc, state, err)
		}
	}
	for _, payload := range []map[string]any{{"bad": make(chan int)}, {"name": 1}} {
		if _, err := Evaluate(spec, payload); err == nil {
			t.Fatal("malformed XML payload accepted")
		}
	}
}
