/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package apicontract

import "strings"

// validatePOSTPolicy binds the supported signature algorithm to its exact wire fields.
func validatePOSTPolicy(method string, request *Request) error {
	policy := request.PostPolicy
	if policy == nil {
		return nil
	}
	if method != "POST" || request.Encoding != "multipart" || policy.Algorithm != "s3-v2" {
		return Invalid("request.post_policy")
	}
	if len(request.Parts) == 0 || !request.Parts[len(request.Parts)-1].File || request.Parts[len(request.Parts)-1].Name != "file" {
		return Invalid("request.post_policy.file")
	}
	seen := map[string]bool{}
	for name, source := range map[string]string{"AWSAccessKeyId": policy.AccessKey, "policy": policy.Policy, "Signature": policy.Signature} {
		parameter, ok := strings.CutPrefix(source, "$param.")
		if !ok || parameter == "" || seen[source] {
			return Invalid("request.post_policy.source")
		}
		seen[source] = true
		found := false
		for _, part := range request.Parts {
			if part.Name == name && part.Source == source && !part.File && !part.Expand {
				found = true
			}
		}
		if !found {
			return Invalid("request.post_policy.parts")
		}
	}
	return nil
}
