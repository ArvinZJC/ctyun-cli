/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package apicontract

import "strings"

// Native declares a storage signing service and bucket/object binding sources.
// Runtime endpoints and credentials are supplied separately from EOP configuration.
type Native struct {
	Versions     []string `json:"versions,omitempty"`
	Metadata     string   `json:"metadata,omitempty"`
	ContentType  string   `json:"content_type,omitempty"`
	PolicyAuth   bool     `json:"policy_auth,omitempty"`
	Subresources []string `json:"subresources,omitempty"`
	Service      string   `json:"service"`
	Addressing   string   `json:"addressing"`
	Bucket       string   `json:"bucket,omitempty"`
	Object       string   `json:"object,omitempty"`
	V2QueryKeys  []string `json:"v2_query_keys,omitempty"`
}

// ValidateNative requires an explicit native route and response contract.
func ValidateNative(native *Native, path string, response *Response) error {
	if native == nil {
		return nil
	}
	seenVersions := map[string]bool{}
	for _, version := range native.Versions {
		if (version != "v2" && version != "v4") || seenVersions[version] {
			return Invalid("native.versions")
		}
		seenVersions[version] = true
	}
	if native.Service != "s3" && native.Service != "sts" && native.Service != "cloudtrail" {
		return Invalid("native.service")
	}
	if path != "/" || response == nil {
		return Invalid("native.route")
	}
	if native.Addressing != "path" && native.Addressing != "virtual" {
		return Invalid("native.addressing")
	}
	if native.Object != "" && native.Bucket == "" {
		return Invalid("native.object")
	}
	for _, source := range []string{native.Metadata, native.ContentType} {
		if source != "" && (!strings.HasPrefix(source, "$param.") || source == "$param.") {
			return Invalid("native.source")
		}
	}
	for _, source := range []string{native.Bucket, native.Object, native.Metadata, native.ContentType} {
		if source != "" && (source == "$arg." || source == "$param." || !strings.HasPrefix(source, "$arg.") && !strings.HasPrefix(source, "$param.")) {
			return Invalid("native.source")
		}
	}
	for _, keys := range [][]string{native.V2QueryKeys, native.Subresources} {
		seen := map[string]bool{}
		for _, key := range keys {
			if key == "" || seen[key] || strings.ContainsAny(key, "&=?# \r\n") {
				return Invalid("native.v2_query_keys")
			}
			seen[key] = true
		}
	}
	return nil
}
