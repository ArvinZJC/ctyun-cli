/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package client

import (
	"strings"
	"testing"
)

// TestFormMembers preserves ordered tags, empty values and exactly one level of URL encoding.
func TestFormMembers(t *testing.T) {
	body := prepareTestBody(t, BodyInput{Encoding: "form", MemberFields: []string{"Tags", "TagKeys"}, Fields: map[string]any{"Action": "TagUser", "Tags": []any{map[string]any{"Key": "a&中", "Value": ""}}, "TagKeys": []any{"a+b", "x"}}})
	data := readPreparedTestBody(t, body)
	want := "Action=TagUser&TagKeys.member.1=a%2Bb&TagKeys.member.2=x&Tags.member.1.Key=a%26%E4%B8%AD&Tags.member.1.Value="
	if string(data) != want {
		t.Fatal(string(data))
	}
	for _, fields := range []map[string]any{
		{"Tags": "bad"}, {"Tags": []any{nil}}, {"Tags": []any{map[string]any{}}}, {"Tags": []any{map[string]any{"a.b": "x"}}}, {"Tags": []any{map[string]any{"Key": []any{}}}}, {"Tags": []any{"x"}, "Tags.member.1": "collision"},
	} {
		if _, err := PrepareBody(BodyInput{Encoding: "form", MemberFields: []string{"Tags"}, Fields: fields}); err == nil {
			t.Fatal(fields, err)
		}
	}
}

// TestIAMDiagnosticRedaction protects passwords and MFA proof codes in form and JSON errors.
func TestIAMDiagnosticRedaction(t *testing.T) {
	for _, value := range []string{`OldPassword=old-secret&NewPassword=new-secret&AuthenticationCode1=123456&AuthenticationCode2=654321`, `{"NewPassword":"new-secret","AuthenticationCode1":"123456"}`} {
		got := redactCredentialFields(value)
		for _, secret := range []string{"old-secret", "new-secret", "123456", "654321"} {
			if strings.Contains(got, secret) {
				t.Fatal(got)
			}
		}
	}
}
