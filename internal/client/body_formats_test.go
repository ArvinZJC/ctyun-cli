/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// preparedBytes independently checks that body metadata matches the transmitted snapshot.
func preparedBytes(t *testing.T, body *PreparedBody) []byte {
	t.Helper()
	reader, err := body.Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if int64(len(data)) != body.Length || hex.EncodeToString(digest[:]) != body.SHA256 {
		t.Fatal("snapshot metadata mismatch")
	}
	return data
}

// TestPreparedEncodings preserves empty values, escaping, file order, and exact XML bytes.
func TestPreparedEncodings(t *testing.T) {
	for _, tc := range []struct {
		input BodyInput
		want  string
	}{
		{BodyInput{Encoding: "json", Fields: map[string]any{"n": json.Number("9007199254740993")}}, `{"n":9007199254740993}`},
		{BodyInput{Encoding: "json"}, ""},
		{BodyInput{Encoding: "xml", Document: "<r>001</r>"}, "<r>001</r>"},
		{BodyInput{Encoding: "form", Fields: map[string]any{"empty": "", "flag": true, "n": json.Number("1.20"), "text": "a&b c"}}, "empty=&flag=true&n=1.20&text=a%26b+c"},
	} {
		body, err := PrepareBody(tc.input)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(preparedBytes(t, body)); got != tc.want {
			t.Fatalf("got=%q want=%q", got, tc.want)
		}
		if err = body.Close(); err != nil {
			t.Fatal(err)
		}
		if err = body.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err = body.Open(); err == nil {
			t.Fatal("closed snapshot reopened")
		}
	}
	path := filepath.Join(t.TempDir(), `a"b.txt`)
	if err := os.WriteFile(path, []byte("file data"), 0600); err != nil {
		t.Fatal(err)
	}
	body, err := PrepareBody(BodyInput{Encoding: "multipart", Parts: []BodyPart{{Name: "empty", Value: ""}, {Name: "file", Value: path, File: true, ContentType: "text/plain"}}})
	if err != nil {
		t.Fatal(err)
	}
	data := preparedBytes(t, body)
	_, params, err := mime.ParseMediaType(body.ContentType)
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(strings.NewReader(string(data)), params["boundary"])
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "empty" {
		t.Fatalf("first part: %v", err)
	}
	first, err := io.ReadAll(part)
	if err != nil || len(first) != 0 {
		t.Fatal("empty part lost")
	}
	part, err = reader.NextPart()
	if err != nil || part.FileName() != filepath.Base(path) {
		t.Fatalf("file part: %v", err)
	}
	content, err := io.ReadAll(part)
	if err != nil || string(content) != "file data" {
		t.Fatal("file bytes lost")
	}
	if _, err = reader.NextPart(); err != io.EOF {
		t.Fatal("unexpected extra part")
	}
	snapshot := body.path
	if err = body.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(snapshot); !os.IsNotExist(err) {
		t.Fatal("snapshot retained")
	}
}

// TestPreparationFailures rejects malformed inputs and cleans partially prepared uploads.
func TestPreparationFailures(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	for _, input := range []BodyInput{
		{Encoding: "invalid"}, {Encoding: "json", Fields: map[string]any{"x": make(chan int)}},
		{Encoding: "xml", Document: "<bad>"}, {Encoding: "form", Fields: map[string]any{"array": []string{"a"}}},
		{Encoding: "file", Document: tmp}, {Encoding: "file", Document: filepath.Join(tmp, "missing")},
		{Encoding: "multipart", Parts: []BodyPart{{Name: "a\n"}}},
		{Encoding: "multipart", Parts: []BodyPart{{Name: "a", ContentType: "bad"}}},
		{Encoding: "multipart", Parts: []BodyPart{{Name: "file", File: true, Value: tmp}}},
	} {
		if body, err := PrepareBody(input); err == nil {
			if closeErr := body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
			t.Errorf("accepted %#v", input)
		}
		files, err := filepath.Glob(filepath.Join(tmp, "ctyun-body-*"))
		if err != nil || len(files) != 0 {
			t.Fatalf("temporary leak: %v %v", files, err)
		}
	}
	t.Setenv("TMPDIR", filepath.Join(tmp, "missing"))
	if _, err := PrepareBody(BodyInput{Encoding: "file"}); err == nil {
		t.Fatal("missing temp directory accepted")
	}
}

// TestFormJSONFieldsEncodesDeclaredCompositeValuesOnce preserves nested data and exact numbers.
func TestFormJSONFieldsEncodesDeclaredCompositeValuesOnce(t *testing.T) {
	body := prepareTestBody(t, BodyInput{Encoding: "form", JSONFields: []string{"statement"}, Fields: map[string]any{"statement": []any{map[string]any{"id": json.Number("9007199254740993"), "value": "a&b+中文"}}, "region": "0001"}})
	data := readPreparedTestBody(t, body)
	values, err := url.ParseQuery(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("statement") != `[{"id":9007199254740993,"value":"a\u0026b+中文"}]` || values.Get("region") != "0001" {
		t.Fatal(values)
	}
	if _, err := PrepareBody(BodyInput{Encoding: "form", JSONFields: []string{"bad"}, Fields: map[string]any{"bad": make(chan int)}}); err == nil {
		t.Fatal("unserializable field accepted")
	}
}

// TestMultipartRejectsCaseInsensitiveDuplicateFields prevents expanded metadata collisions.
func TestMultipartRejectsCaseInsensitiveDuplicateFields(t *testing.T) {
	body, err := PrepareBody(BodyInput{Encoding: "multipart", Parts: []BodyPart{{Name: "X-Test", Value: "one"}, {Name: "x-test", Value: "two"}}})
	if body != nil {
		if closeErr := body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}
	if err == nil {
		t.Fatal("duplicate field accepted")
	}
}
