/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package durationlimit checks whole-second settings before conversion to time.Duration.
package durationlimit

import (
	"math"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// CheckSeconds reports positive overflow. Callers define their own zero and negative semantics.
func CheckSeconds(seconds int, field string) error {
	const maximum = math.MaxInt64 / int64(time.Second)
	if int64(seconds) > maximum {
		return diagnostic.New("error.seconds_exceed_maximum", field, maximum)
	}
	return nil
}
