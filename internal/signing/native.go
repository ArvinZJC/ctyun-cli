/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package signing

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

// NativeOptions supplies explicit storage signing scope and immutable payload identity.
// Resource is the decoded V2 bucket/object resource including the bucket for either addressing style.
// Signing escapes this resource without normalising object keys.
// QueryKeys lists V2 subresources; ordinary listing filters do not participate in V2 signing.
type NativeOptions struct {
	Version     string
	Region      string
	Service     string
	Resource    string
	QueryKeys   []string
	PayloadHash string
	Token       string
	Now         time.Time
}

// NativePath escapes UTF-8 bytes according to the storage protocol without cleaning paths.
func NativePath(value string) string {
	const digits = "0123456789ABCDEF"
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune("-_.~/", rune(b)) {
			out.WriteByte(b)
		} else {
			out.WriteByte('%')
			out.WriteByte(digits[b>>4])
			out.WriteByte(digits[b&15])
		}
	}
	return out.String()
}

// nativeQueryValue applies URI escaping to a query component, including slashes.
func nativeQueryValue(value string) string { return strings.ReplaceAll(NativePath(value), "/", "%2F") }

// NativeQuery sorts encoded names and values and retains empty and repeated parameters.
func NativeQuery(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", apicontract.Invalid("native.query")
	}
	var pairs [][2]string
	for key, items := range values {
		for _, value := range items {
			pairs = append(pairs, [2]string{nativeQueryValue(key), nativeQueryValue(value)})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] == pairs[j][0] {
			return pairs[i][1] < pairs[j][1]
		}
		return pairs[i][0] < pairs[j][0]
	})
	parts := make([]string, len(pairs))
	for i, pair := range pairs {
		parts[i] = pair[0] + "=" + pair[1]
	}
	return strings.Join(parts, "&"), nil
}

// SignNative adds storage V2 or V4 authorization without reading or changing the request body.
func SignNative(req *http.Request, options NativeOptions, credentials config.Credentials) error {
	if credentials.AccessKey == "" || credentials.SecretKey == "" {
		return apicontract.Invalid("native.credentials")
	}
	if options.Version != "v2" && options.Version != "v4" {
		return apicontract.Invalid("native.version")
	}
	if options.Now.IsZero() {
		return apicontract.Invalid("native.date")
	}
	query, err := NativeQuery(req.URL.RawQuery)
	if err != nil {
		return err
	}
	req.URL.RawQuery = query
	req.URL.RawPath = NativePath(req.URL.Path)
	if options.Token != "" {
		req.Header.Set("X-Amz-Security-Token", options.Token)
	}
	if options.Version == "v2" {
		return signNativeV2(req, options, credentials)
	}
	if options.Region == "" || options.Service == "" || strings.ContainsAny(options.Region+options.Service, "/\r\n \t") {
		return apicontract.Invalid("native.scope")
	}
	digest, err := hex.DecodeString(options.PayloadHash)
	if err != nil || len(digest) != sha256.Size {
		return apicontract.Invalid("native.payload_hash")
	}
	date := options.Now.UTC().Format("20060102T150405Z")
	req.Header.Set("X-Amz-Date", date)
	req.Header.Set("X-Amz-Content-Sha256", options.PayloadHash)
	headers, names := nativeHeaders(req, false)
	scope := date[:8] + "/" + options.Region + "/" + options.Service + "/aws4_request"
	path := req.URL.EscapedPath()
	if path == "" {
		path = "/"
	}
	canonical := req.Method + "\n" + path + "\n" + query + "\n" + headers + "\n" + names + "\n" + options.PayloadHash
	hash := sha256.Sum256([]byte(canonical))
	signingText := "AWS4-HMAC-SHA256\n" + date + "\n" + scope + "\n" + hex.EncodeToString(hash[:])
	key := hmacSHA256(date[:8], "AWS4"+credentials.SecretKey)
	key = hmacSHA256(options.Region, string(key))
	key = hmacSHA256(options.Service, string(key))
	key = hmacSHA256("aws4_request", string(key))
	signature := hex.EncodeToString(hmacSHA256(signingText, string(key)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+credentials.AccessKey+"/"+scope+", SignedHeaders="+names+", Signature="+signature)
	return nil
}

// nativeHeaders canonicalises stable headers while excluding transport-owned headers.
func nativeHeaders(req *http.Request, amzOnly bool) (string, string) {
	values := map[string][]string{}
	for key, items := range req.Header {
		key = strings.ToLower(key)
		if amzOnly && !strings.HasPrefix(key, "x-amz-") {
			continue
		}
		switch key {
		case "authorization", "user-agent", "connection", "transfer-encoding", "expect", "accept-encoding", "host":
			continue
		}
		for _, value := range items {
			values[key] = append(values[key], strings.Join(strings.Fields(value), " "))
		}
	}
	if !amzOnly {
		host := req.Host
		if host == "" {
			host = req.URL.Host
		}
		values["host"] = []string{host}
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonical strings.Builder
	for _, name := range names {
		canonical.WriteString(name + ":" + strings.Join(values[name], ",") + "\n")
	}
	return canonical.String(), strings.Join(names, ";")
}

// signNativeV2 signs escaped CTyun resources and only explicitly declared subresources.
func signNativeV2(req *http.Request, options NativeOptions, credentials config.Credentials) error {
	if !strings.HasPrefix(options.Resource, "/") {
		return apicontract.Invalid("native.resource")
	}
	resource := NativePath(options.Resource)
	values := req.URL.Query()
	var pairs []string
	keys := append([]string(nil), options.QueryKeys...)
	sort.Strings(keys)
	for _, key := range keys {
		sort.Strings(values[key])
		for _, value := range values[key] {
			part := key
			if value != "" {
				part += "=" + value
			}
			pairs = append(pairs, part)
		}
	}
	if len(pairs) > 0 {
		resource += "?" + strings.Join(pairs, "&")
	}
	date := options.Now.UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	if req.Header.Get("X-Amz-Date") != "" {
		date = ""
	}
	headers, _ := nativeHeaders(req, true)
	canonical := req.Method + "\n" + req.Header.Get("Content-Md5") + "\n" + req.Header.Get("Content-Type") + "\n" + date + "\n" + headers + resource
	req.Header.Set("Authorization", "AWS "+credentials.AccessKey+":"+SignPOSTPolicyV2(canonical, credentials.SecretKey))
	return nil
}
