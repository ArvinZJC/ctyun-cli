/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
)

// NativeRequest specifies storage routing independently of EOP profiles and headers.
type NativeRequest struct {
	Version    string
	Region     string
	Service    string
	Addressing string
	Bucket     string
	Object     string
	Token      string
	QueryKeys  []string
}

// nativeBucketName permits DNS-compatible bucket labels without host or path delimiters.
var nativeBucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)

// nativeURL constructs the wire URL and decoded V2 resource without normalising object keys.
func nativeURL(endpoint string, native NativeRequest) (string, string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return "", "", apicontract.Invalid("native.endpoint")
	}
	if native.Addressing != "path" && native.Addressing != "virtual" {
		return "", "", apicontract.Invalid("native.addressing")
	}
	if native.Bucket != "" && (len(native.Bucket) < 3 || len(native.Bucket) > 63 || !nativeBucketName.MatchString(native.Bucket) || strings.Contains(native.Bucket, "..") || strings.Contains(native.Bucket, ".-") || strings.Contains(native.Bucket, "-.") || net.ParseIP(native.Bucket) != nil) {
		return "", "", apicontract.Invalid("native.bucket")
	}
	if native.Object != "" && native.Bucket == "" {
		return "", "", apicontract.Invalid("native.object")
	}
	path := "/"
	resource := "/"
	if native.Bucket != "" {
		resource += native.Bucket + "/" + native.Object
		if native.Addressing == "virtual" {
			if net.ParseIP(parsed.Hostname()) != nil {
				return "", "", apicontract.Invalid("native.endpoint")
			}
			parsed.Host = native.Bucket + "." + parsed.Host
			path += native.Object
		} else {
			path += native.Bucket + "/" + native.Object
		}
	}
	parsed.Path = path
	parsed.RawPath = signing.NativePath(path)
	return parsed.String(), resource, nil
}
