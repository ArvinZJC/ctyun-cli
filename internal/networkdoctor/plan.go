/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

// SourceInput describes one independently verified hosted metadata capability.
type SourceInput struct {
	Capability    string
	Subject       string
	Source        distribution.Source
	IndexName     string
	SignatureName string
}

// Input contains resolved, credential-free values used to build a plan.
type Input struct {
	Sources        []SourceInput
	CTyunEndpoints []string
	PublicKey      string
	Proxy          func(*http.Request) (*url.URL, error)
}

// Build constructs a deterministic dependency-aware diagnostic plan.
func Build(input Input) (Plan, error) {
	builder := planBuilder{
		input:        input,
		routeIDs:     make(map[string]string),
		httpsIDs:     make(map[string]string),
		capabilities: make(map[string][]string),
	}
	if len(input.Sources) > 0 {
		builder.add(Check{ID: "configuration-public-key", Kind: CheckConfiguration, Subject: "public-key", Target: "release public key"})
	}
	for _, source := range input.Sources {
		if err := builder.addSource(source); err != nil {
			return Plan{}, err
		}
	}
	if err := builder.addCTyunEndpoints(input.CTyunEndpoints); err != nil {
		return Plan{}, err
	}
	builder.addCapabilities()
	return Plan{Checks: builder.checks, Capabilities: builder.capabilityList, PublicKey: input.PublicKey}, nil
}

// planBuilder accumulates unique checks and capability alternatives in order.
type planBuilder struct {
	input          Input
	checks         []Check
	routeIDs       map[string]string
	httpsIDs       map[string]string
	capabilities   map[string][]string
	capabilityList []Capability
}

// addSource adds one signed-index chain for each selected source candidate.
func (builder *planBuilder) addSource(input SourceInput) error {
	candidates := append([]distribution.Source{input.Source}, input.Source.Fallbacks...)
	for index, candidate := range candidates {
		baseURL, target, err := safeHostedURL(candidate.URL)
		if err != nil {
			return err
		}
		role := "fallback"
		if index == 0 {
			role = "primary"
		}
		routeID, err := builder.ensureRoute(baseURL, input.Subject)
		if err != nil {
			return err
		}
		httpsID := builder.ensureHTTPS(baseURL, target, input.Subject, routeID)
		prefix := stableID(input.Subject, candidate.Name, fmt.Sprint(index))
		indexID := prefix + "-index"
		signatureID := prefix + "-signature"
		verificationID := prefix + "-verification"
		builder.add(Check{ID: indexID, Kind: CheckIndex, Capability: input.Capability, Subject: input.Subject, SourceName: candidate.Name, Role: role, Target: target, RequestURL: distribution.JoinURL(baseURL, input.IndexName), DependsOn: []string{httpsID}, Alternative: len(candidates) > 1})
		builder.add(Check{ID: signatureID, Kind: CheckSignature, Capability: input.Capability, Subject: input.Subject, SourceName: candidate.Name, Role: role, Target: target, RequestURL: distribution.JoinURL(baseURL, input.SignatureName), DependsOn: []string{httpsID}, Alternative: len(candidates) > 1})
		builder.add(Check{ID: verificationID, Kind: CheckVerification, Capability: input.Capability, Subject: input.Subject, SourceName: candidate.Name, Role: role, Target: target, DependsOn: []string{"configuration-public-key", indexID, signatureID}, Alternative: len(candidates) > 1})
		builder.capabilities[input.Capability] = append(builder.capabilities[input.Capability], verificationID)
	}
	return nil
}

// addCTyunEndpoints adds one route and HTTPS chain per unique endpoint origin.
func (builder *planBuilder) addCTyunEndpoints(endpoints []string) error {
	seen := make(map[string]bool)
	for _, endpoint := range endpoints {
		baseURL, target, err := safeHostedURL(endpoint)
		if err != nil {
			return err
		}
		if seen[target] {
			continue
		}
		seen[target] = true
		routeID, err := builder.ensureRoute(baseURL, "ctyun")
		if err != nil {
			return err
		}
		httpsID := builder.ensureHTTPSForSubject(baseURL, target, "ctyun", routeID)
		capabilityID := "ctyun-" + stableID(target)
		builder.capabilities[capabilityID] = []string{httpsID}
	}
	return nil
}

// ensureRoute returns a shared route check for a direct or proxied origin.
func (builder *planBuilder) ensureRoute(rawURL, subject string) (string, error) {
	routeURL := rawURL
	role := "direct"
	request, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return "", err
	}
	if builder.input.Proxy != nil {
		proxyURL, proxyErr := builder.input.Proxy(request)
		if proxyErr != nil {
			return "", diagnostic.Wrap("error.doctor_proxy", proxyErr)
		}
		if proxyURL != nil {
			routeURL = proxyURL.String()
			role = "proxy"
		}
	}
	_, target, err := safeHostedURL(routeURL)
	if err != nil {
		return "", err
	}
	if id := builder.routeIDs[target]; id != "" {
		return id, nil
	}
	id := fmt.Sprintf("route-%d", len(builder.routeIDs))
	builder.routeIDs[target] = id
	builder.add(Check{ID: id, Kind: CheckRoute, Subject: subject, Role: role, Target: target, RequestURL: target})
	return id, nil
}

// ensureHTTPS returns a shared HTTPS check for one subject and target.
func (builder *planBuilder) ensureHTTPS(rawURL, target, subject, routeID string) string {
	key := subject + "\x00" + target
	if id := builder.httpsIDs[key]; id != "" {
		return id
	}
	return builder.ensureHTTPSForSubject(rawURL, target, subject, routeID)
}

// ensureHTTPSForSubject creates an HTTPS check unless the subject already has one.
func (builder *planBuilder) ensureHTTPSForSubject(rawURL, target, subject, routeID string) string {
	key := subject + "\x00" + target
	if id := builder.httpsIDs[key]; id != "" {
		return id
	}
	id := fmt.Sprintf("https-%d", len(builder.httpsIDs))
	builder.httpsIDs[key] = id
	builder.add(Check{ID: id, Kind: CheckHTTPS, Subject: subject, Target: target, RequestURL: rawURL, DependsOn: []string{routeID}})
	return id
}

// addCapabilities appends deterministic required-capability result checks.
func (builder *planBuilder) addCapabilities() {
	ids := make([]string, 0, len(builder.capabilities))
	for id := range builder.capabilities {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		alternatives := builder.capabilities[id]
		builder.capabilityList = append(builder.capabilityList, Capability{ID: id, Alternatives: slices.Clone(alternatives), Required: true})
		builder.add(Check{ID: "capability-" + stableID(id), Kind: CheckCapability, Capability: id, Subject: id, Target: id, DependsOn: slices.Clone(alternatives)})
	}
}

// add appends a check and assigns its immutable sequence number.
func (builder *planBuilder) add(check Check) {
	check.Sequence = len(builder.checks)
	builder.checks = append(builder.checks, check)
}

// safeHostedURL returns a query-free HTTPS request base and display origin.
func safeHostedURL(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", "", diagnostic.New("error.doctor_invalid_endpoint", safeInvalidTarget(raw))
	}
	base := &url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: parsed.Path, RawPath: parsed.RawPath}
	target := (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
	return base.String(), target, nil
}

// safeInvalidTarget reduces an invalid URL to a non-sensitive diagnostic label.
func safeInvalidTarget(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return "invalid endpoint"
}

// stableID converts machine-oriented plan identifiers to a safe stable token.
func stableID(parts ...string) string {
	value := strings.ToLower(strings.Join(parts, "-"))
	var result strings.Builder
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' {
			result.WriteRune(character)
		} else {
			result.WriteByte('-')
		}
	}
	return strings.Trim(result.String(), "-")
}
