/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// validateOperationPathBindings requires every HTTP path placeholder to name
// an argument declared by its command, so no unresolved template can be sent.
func validateOperationPathBindings(command Command, operation Operation) error {
	for segment := range strings.SplitSeq(operation.Path, "/") {
		if !strings.ContainsAny(segment, "{}") {
			continue
		}
		if !strings.HasPrefix(segment, "{") || !validCommandPathSegment(segment) {
			return diagnostic.New("error.operation_invalid_path", command.Operation, operation.Path)
		}
		if !slices.Contains(command.Path, segment) {
			return diagnostic.New("error.missing_path_argument", strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"))
		}
	}
	return nil
}
