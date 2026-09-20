/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"encoding/base64"
	"encoding/json"
	"maps"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
)

// resolvePOSTPolicy signs supplied policy bytes only when no external signature exists.
// The storage secret is read separately from gateway credentials and never enters metadata.
func resolvePOSTPolicy(request *apicontract.Request, values map[string]string, getenv func(string) string) (map[string]string, error) {
	if request == nil || request.PostPolicy == nil {
		return values, nil
	}
	contract := request.PostPolicy
	ak := values[strings.TrimPrefix(contract.AccessKey, "$param.")]
	policy := values[strings.TrimPrefix(contract.Policy, "$param.")]
	signatureName := strings.TrimPrefix(contract.Signature, "$param.")
	signature := values[signatureName]
	if ak == "" && policy == "" && signature == "" {
		return values, nil
	}
	if ak == "" || policy == "" {
		return nil, diagnostic.New("error.post_policy_credentials")
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(policy)
	var document map[string]json.RawMessage
	if err != nil || json.Unmarshal(decoded, &document) != nil || document == nil {
		return nil, diagnostic.New("error.post_policy_document")
	}
	if signature != "" {
		return values, nil
	}
	secret := getenv("CTYUN_STORAGE_SK")
	if secret == "" {
		return nil, diagnostic.New("error.post_policy_secret")
	}
	signed := maps.Clone(values)
	signed[signatureName] = signing.SignPOSTPolicyV2(policy, secret)
	return signed, nil
}
