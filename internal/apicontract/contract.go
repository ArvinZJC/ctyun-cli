/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package apicontract defines shared HTTP representation and success contracts.
package apicontract

import (
	"mime"
	"net/http"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// Request selects an explicit body encoder; sources use plugin binding syntax.
type Request struct {
	PostPolicy   *PostPolicy `json:"post_policy,omitempty"`
	Encoding     string      `json:"encoding"`
	MemberFields []string    `json:"member_fields,omitempty"`
	JSONFields   []string    `json:"json_fields,omitempty"`
	Document     string      `json:"document,omitempty"`
	Parts        []Part      `json:"parts,omitempty"`
}

// PostPolicy identifies multipart inputs used for independent storage policy signing.
// Sources refer to scalar command parameters; EOP credentials are never reused.
type PostPolicy struct {
	Algorithm string `json:"algorithm"`
	AccessKey string `json:"access_key"`
	Policy    string `json:"policy"`
	Signature string `json:"signature"`
}

// Part declares one ordered multipart field, optionally sourced from a local file.
type Part struct {
	Expand      bool   `json:"expand,omitempty"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	File        bool   `json:"file,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// Response declares successful HTTP representations and conjunctive JSON checks.
type Response struct {
	Variants []Variant `json:"variants"`
	Success  []Check   `json:"success,omitempty"`
}

// Variant defines one successful status and its body and header representation.
type Variant struct {
	XMLRoot   *XMLName          `json:"xml_root,omitempty"`
	Status    int               `json:"status"`
	Format    string            `json:"format"`
	MediaType string            `json:"media_type,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// Check requires a scalar JSON path to match one of the allowed success values.
type Check struct {
	Path   string   `json:"path"`
	Values []string `json:"values"`
}

// SupportedMethod is the shared catalog and runtime HTTP method vocabulary.
func SupportedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// Invalid identifies a malformed transport field without embedding English prose.
func Invalid(field string) error { return diagnostic.New("error.invalid_transport", field) }

// Validate rejects ambiguous representations and impossible HTTP response bodies.
func Validate(method, contentType string, request *Request, response *Response) error {
	if !SupportedMethod(method) {
		return Invalid("method")
	}
	if request != nil {
		if err := validatePOSTPolicy(method, request); err != nil {
			return err
		}
		expected := ""
		switch request.Encoding {
		case "json":
			expected = "application/json"
		case "xml":
			expected = "application/xml"
		case "form":
			expected = "application/x-www-form-urlencoded"
		case "multipart":
			expected = "multipart/form-data"
		case "file":
		default:
			return Invalid("request.encoding")
		}
		if (request.Encoding == "xml" || request.Encoding == "file") != (request.Document != "") {
			return Invalid("request.document")
		}
		if (request.Encoding == "multipart") != (len(request.Parts) > 0) {
			return Invalid("request.parts")
		}
		if contentType != "" {
			media, params, err := mime.ParseMediaType(contentType)
			if err != nil || (expected != "" && media != expected && !(request.Encoding == "xml" && media == "text/xml") && !(request.Encoding == "form" && media == "application/octet-stream")) {
				return Invalid("content_type")
			}
			if request.Encoding == "multipart" && len(params) != 0 {
				return Invalid("content_type")
			}
		}
		fields := map[string]bool{}
		for _, field := range request.JSONFields {
			if request.Encoding != "form" || field == "" || fields[field] {
				return Invalid("request.json_fields")
			}
			fields[field] = true
		}
		for _, field := range request.MemberFields {
			if request.Encoding != "form" || field == "" || fields[field] {
				return Invalid("request.member_fields")
			}
			fields[field] = true
		}
		seen := map[string]bool{}
		for _, part := range request.Parts {
			if part.Name == "" || strings.ContainsAny(part.Name, "\r\n\x00") || seen[part.Name] || part.Source == "" {
				return Invalid("request.parts")
			}
			if part.Expand && (part.File || part.ContentType != "" || !HeaderName(part.Name)) {
				return Invalid("request.parts.expand")
			}
			seen[part.Name] = true
			if part.ContentType != "" {
				if media, _, err := mime.ParseMediaType(part.ContentType); err != nil || !strings.Contains(media, "/") || strings.ContainsAny(part.ContentType, "\r\n") {
					return Invalid("request.parts.content_type")
				}
			}
		}
	}
	if response == nil {
		if method == http.MethodHead || method == http.MethodOptions {
			return Invalid("response")
		}
		return nil
	}
	if len(response.Variants) == 0 {
		return Invalid("response.variants")
	}
	seen := map[int]bool{}
	for _, variant := range response.Variants {
		if variant.MediaType != "" {
			mediaType, params, err := mime.ParseMediaType(variant.MediaType)
			if err != nil || len(params) != 0 || !strings.EqualFold(mediaType, variant.MediaType) || !strings.Contains(mediaType, "/") || strings.ContainsAny(mediaType, "*\r\n") || strings.ContainsAny(variant.MediaType, "\r\n") {
				return Invalid("response.media_type")
			}
		}
		if seen[variant.Status] || (variant.Status < 200 || variant.Status >= 300) && variant.Status != 304 && variant.Status != 303 {
			return Invalid("response.status")
		}
		if variant.Status == 303 {
			location := false
			for _, header := range variant.Headers {
				location = location || strings.EqualFold(header, "Location")
			}
			if method != "POST" || variant.Format != "empty" || !location {
				return Invalid("response.redirect")
			}
		}
		seen[variant.Status] = true
		switch variant.Format {
		case "json", "xml", "text", "binary", "empty":
		default:
			return Invalid("response.format")
		}
		if (method == http.MethodHead || variant.Status == 204 || variant.Status == 304 || variant.Status == 205) && variant.Format != "empty" {
			return Invalid("response.format")
		}
		if variant.XMLRoot != nil && (variant.Format != "xml" || variant.XMLRoot.Local == "") {
			return Invalid("response.xml_root")
		}
		if variant.Format != "json" && len(response.Success) != 0 {
			return Invalid("response.success")
		}
		for key, header := range variant.Headers {
			if variant.Format != "empty" || key == "" || !HeaderName(header) {
				return Invalid("response.headers")
			}
		}
	}
	for _, check := range response.Success {
		if check.Path == "" || len(check.Values) == 0 {
			return Invalid("response.success")
		}
		for part := range strings.SplitSeq(check.Path, ".") {
			if strings.TrimSpace(part) == "" {
				return Invalid("response.success.path")
			}
		}
		if slices.Contains(check.Values, "") {
			return Invalid("response.success.values")
		}
	}
	return nil
}

// HeaderName reports whether name is an RFC HTTP field-name token.
func HeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	return true
}
