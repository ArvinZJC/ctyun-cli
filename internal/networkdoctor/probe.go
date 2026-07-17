/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

const (
	maxIndexBytes     int64 = 4 << 20
	maxSignatureBytes int64 = 16 << 10
)

// Resolver resolves route host names for diagnostic checks.
type Resolver interface {
	LookupHost(context.Context, string) ([]string, error)
}

// Dependencies provides replaceable network and clock behavior.
type Dependencies struct {
	Resolver  Resolver
	Transport http.RoundTripper
	Proxy     func(*http.Request) (*url.URL, error)
	Now       func() time.Time
}

// probeOutput keeps bounded verification bytes internal to runner waves.
type probeOutput struct {
	result Result
	data   []byte
}

// executeCheck performs one planned check and returns a structured result.
func executeCheck(ctx context.Context, check Check, plan Plan, dependencies Dependencies, data map[string][]byte) probeOutput {
	started := dependencies.Now()
	result := Result{Check: check, Status: StatusPassed}
	var payload []byte
	err := ctx.Err()
	if err != nil {
		result.Duration = dependencies.Now().Sub(started)
		result.Status = StatusFailed
		result.Category = classifyProbeError(err, check)
		result.DetailKey = "doctor.detail." + string(result.Category)
		result.Err = err
		return probeOutput{result: result}
	}
	switch check.Kind {
	case CheckConfiguration:
		if plan.PublicKey == "" {
			err = errors.New("trusted release public key is unavailable")
			result.Category = FailurePublicKey
		}
	case CheckRoute:
		err = probeRoute(ctx, check, dependencies)
	case CheckHTTPS:
		err = probeHTTPS(ctx, check, dependencies)
	case CheckIndex:
		payload, err = distribution.HTTPGetBytesContext(ctx, check.RequestURL, dependencies.Transport, maxIndexBytes)
	case CheckSignature:
		payload, err = distribution.HTTPGetBytesContext(ctx, check.RequestURL, dependencies.Transport, maxSignatureBytes)
	case CheckVerification:
		if len(check.DependsOn) >= 3 {
			err = distribution.VerifyIndexSignature(data[check.DependsOn[1]], data[check.DependsOn[2]], plan.PublicKey, check.Subject)
		}
	}
	result.Duration = dependencies.Now().Sub(started)
	if err != nil {
		result.Status = StatusFailed
		if result.Category == "" {
			result.Category = classifyProbeError(err, check)
		}
		result.DetailKey = "doctor.detail." + string(result.Category)
		result.Err = err
	} else {
		result.DetailKey = "doctor.detail.passed"
	}
	return probeOutput{result: result, data: payload}
}

// probeRoute resolves the direct destination or selected proxy route host.
func probeRoute(ctx context.Context, check Check, dependencies Dependencies) error {
	parsed, err := url.Parse(check.RequestURL)
	if err != nil {
		return err
	}
	_, err = dependencies.Resolver.LookupHost(ctx, parsed.Hostname())
	return err
}

// probeHTTPS proves safe unauthenticated HTTPS reachability for any status.
func probeHTTPS(ctx context.Context, check Check, dependencies Dependencies) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, check.RequestURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Transport: dependencies.Transport}
	client.CheckRedirect = func(next *http.Request, previous []*http.Request) error {
		if next.URL.Scheme != "https" {
			return distribution.ErrUnsafeRedirect
		}
		if len(previous) >= 10 {
			return distribution.ErrUnsafeRedirect
		}
		return nil
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := response.Body.Close(); err == nil {
			err = closeErr
		}
	}()
	_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	return err
}

// classifyProbeError maps wrapped transport failures to stable diagnostic layers.
func classifyProbeError(err error, check Check) FailureCategory {
	if errors.Is(err, context.Canceled) {
		return FailureCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureTimeout
	}
	if errors.Is(err, distribution.ErrUnsafeRedirect) {
		return FailureRedirect
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return FailureDNS
	}
	var unknownAuthority x509.UnknownAuthorityError
	var certificateInvalid x509.CertificateInvalidError
	var recordHeader tls.RecordHeaderError
	if errors.As(err, &unknownAuthority) || errors.As(err, &certificateInvalid) || errors.As(err, &recordHeader) {
		return FailureTLS
	}
	if check.Kind == CheckRoute && check.Role == "proxy" {
		return FailureProxy
	}
	switch check.Kind {
	case CheckIndex:
		return FailureIndex
	case CheckSignature, CheckVerification:
		return FailureSignature
	case CheckConfiguration:
		return FailureConfiguration
	default:
		return FailureConnection
	}
}
