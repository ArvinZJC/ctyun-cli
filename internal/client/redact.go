/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import "regexp"

// sensitiveField matches credential fields used by OpenAPI and storage responses.
const sensitiveField = `(?:x[-_]amz[-_])?(?:policy|credential|signature|access[-_]?key(?:[-_]?id)?|awsaccesskeyid|secret[-_]?access[-_]?key|secret[-_]?key|session[-_]?token|security[-_]?token|authorization|password|base32stringseed|qrcodepng)`

// sensitiveFieldName recognises complete multipart names whose values must not enter diagnostics.
var sensitiveFieldName = regexp.MustCompile(`(?i)^` + sensitiveField + `$`)

// credentialPatterns redact unknown returned secrets as well as known signing inputs.
var credentialPatterns = []struct {
	expression  *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)("` + sensitiveField + `"\s*:\s*)"(?:\\.|[^"\\])*"`), `${1}"[REDACTED]"`},
	{regexp.MustCompile(`(?i)(\b` + sensitiveField + `=)[^&\s<>"']*`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(?i)(<(?:[\w.-]+:)?` + sensitiveField + `(?:\s[^>]*)?>)[^<]*`), `${1}[REDACTED]`},
	{regexp.MustCompile(`(://)[^/@\s]+@`), `${1}[REDACTED]@`},
}

// redactCredentialFields removes common JSON, XML, form, and URL credential values.
func redactCredentialFields(value string) string {
	for _, pattern := range credentialPatterns {
		value = pattern.expression.ReplaceAllString(value, pattern.replacement)
	}
	return value
}
