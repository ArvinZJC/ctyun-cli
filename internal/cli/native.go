/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"mime"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// nativeRequest resolves storage-only environment settings and declared resource bindings.
func nativeRequest(contract *apicontract.Native, args, parameters map[string]string, getenv func(string) string) (*client.NativeRequest, string, config.Credentials, error) {
	creds := config.Credentials{AccessKey: getenv("CTYUN_STORAGE_AK"), SecretKey: getenv("CTYUN_STORAGE_SK"), AccessKeySource: "env", SecretKeySource: "env"}
	endpoint := getenv("CTYUN_STORAGE_ENDPOINT")
	version := getenv("CTYUN_STORAGE_SIGNATURE_VERSION")
	if version == "" {
		version = "v4"
	}
	if version != "v2" && version != "v4" || len(contract.Versions) != 0 && !slices.Contains(contract.Versions, version) {
		return nil, "", creds, apicontract.Invalid("native.version")
	}
	region := getenv("CTYUN_STORAGE_REGION")
	required := []struct{ name, value string }{{"CTYUN_STORAGE_AK", creds.AccessKey}, {"CTYUN_STORAGE_SK", creds.SecretKey}, {"CTYUN_STORAGE_ENDPOINT", endpoint}}
	if contract.PolicyAuth {
		required = required[2:]
		version = "post-policy"
	}
	for _, item := range required {
		if item.value == "" {
			return nil, "", creds, diagnostic.New("error.native_setting_required", item.name)
		}
	}
	if version == "v4" && region == "" {
		return nil, "", creds, diagnostic.New("error.native_setting_required", "CTYUN_STORAGE_REGION")
	}
	resolve := func(source string) string {
		if name, ok := strings.CutPrefix(source, "$arg."); ok {
			return args[name]
		}
		return parameters[strings.TrimPrefix(source, "$param.")]
	}
	native := &client.NativeRequest{Version: version, Region: region, Service: contract.Service, Addressing: contract.Addressing, Bucket: resolve(contract.Bucket), Object: resolve(contract.Object), Token: getenv("CTYUN_STORAGE_SECURITY_TOKEN"), QueryKeys: contract.V2QueryKeys}
	if contract.Bucket != "" && native.Bucket == "" || contract.Object != "" && native.Object == "" {
		return nil, "", creds, apicontract.Invalid("native.source")
	}
	return native, endpoint, creds, nil
}

// nativeRequestHeaders resolves MIME type and object metadata without adding JSON body fields.
func nativeRequestHeaders(contract *apicontract.Native, parameters map[string]string, headers map[string]string) (string, error) {
	contentType := parameters[strings.TrimPrefix(contract.ContentType, "$param.")]
	if contract.ContentType != "" && contentType != "" {
		if media, _, err := mime.ParseMediaType(contentType); err != nil || !strings.Contains(media, "/") || strings.ContainsAny(contentType, "\r\n\x00") {
			return "", apicontract.Invalid("native.content_type")
		}
	}
	raw := parameters[strings.TrimPrefix(contract.Metadata, "$param.")]
	if contract.Metadata != "" && raw != "" {
		parts, err := expandMultipartPart("x-amz-meta-", raw)
		if err != nil {
			return "", err
		}
		for _, part := range parts {
			headers[part.Name] = part.Value
		}
	}
	return contentType, nil
}
