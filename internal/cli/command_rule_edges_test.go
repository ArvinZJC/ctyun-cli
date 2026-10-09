/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// documentFailureReader injects input failures while tracking resource release.
type documentFailureReader struct {
	io.Reader
	readErr  error
	closeErr error
	closes   int
}

// Read returns the injected failure before consuming document bytes.
func (reader *documentFailureReader) Read(data []byte) (int, error) {
	if reader.readErr != nil {
		return 0, reader.readErr
	}
	return reader.Reader.Read(data)
}

// Close records each release attempt and returns the injected close failure.
func (reader *documentFailureReader) Close() error {
	reader.closes++
	return reader.closeErr
}

// TestXMLDocumentInputFailuresPreserveCauseAndClose releases input after every read and preserves the primary failure.
func TestXMLDocumentInputFailuresPreserveCauseAndClose(t *testing.T) {
	original := openCommandDocument
	t.Cleanup(func() { openCommandDocument = original })
	readErr := errors.New("document read failure")
	closeErr := errors.New("document close failure")
	operation := plugin.Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$param.document"}}
	command := plugin.Command{Parameters: []plugin.Parameter{{Name: "document", Input: "file"}}}
	for _, tc := range []struct {
		name                    string
		readErr, closeErr, want error
	}{
		{"read", readErr, nil, readErr},
		{"close", nil, closeErr, closeErr},
		{"read precedes close", readErr, closeErr, readErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &documentFailureReader{Reader: strings.NewReader("<root/>"), readErr: tc.readErr, closeErr: tc.closeErr}
			openCommandDocument = func(path string) (io.ReadCloser, error) {
				if path != "document.xml" {
					t.Fatalf("input path = %q", path)
				}
				return reader, nil
			}
			body, err := prepareCommandBody(operation, command, nil, map[string]string{"document": "document.xml"}, coreconfig.Profile{}, nil)
			if !errors.Is(err, tc.want) || body != nil || reader.closes != 1 {
				t.Fatalf("body=%v error=%v close count=%d", body, err, reader.closes)
			}
		})
	}
}

// TestUnboundTypedOptionDoesNotCreateRequestBody keeps query-only inputs out of an undeclared body.
func TestUnboundTypedOptionDoesNotCreateRequestBody(t *testing.T) {
	parameters := []plugin.Parameter{{Name: "count", Flag: "count", Target: "count", ValueType: plugin.ParameterValueInteger}}
	body, err := resolveRequestBody(nil, coreconfig.Profile{}, nil, map[string]string{"count": "2"}, parameters, "en-US")
	if err != nil || len(body) != 0 {
		t.Fatalf("undeclared body = %v, error = %v", body, err)
	}
}

// TestConditionalAnyOfHelpUsesCorrectOptionRoles preserves the selector and alternatives in each localized help row.
func TestConditionalAnyOfHelpUsesCorrectOptionRoles(t *testing.T) {
	command := plugin.Command{ID: "sample.restore", Parameters: []plugin.Parameter{
		{Name: "mode", Flag: "mode"},
		{Name: "id", Flag: "id", Description: "Resource identifier"},
		{Name: "name", Flag: "name"},
	}, ConditionalRequirements: []plugin.ConditionalRequirement{{
		When:  plugin.ParameterCondition{Parameter: "mode", Equals: "restore"},
		AnyOf: []string{"id", "name"},
	}}}
	for language, want := range map[string]string{
		"en-US": "Resource identifier (at least one of --id, --name required when --mode is restore)",
		"en-GB": "Resource identifier (at least one of --id, --name required when --mode is restore)",
		"zh-CN": "Resource identifier (当 --mode 为 restore 时，--id, --name 至少填一项)",
	} {
		if got := parameterHelpDescription(plugin.Bundle{}, command, command.Parameters[1], language); got != want {
			t.Errorf("%s conditional help = %q, want %q", language, got, want)
		}
	}
}
