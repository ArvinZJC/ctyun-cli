/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package durationlimit

import (
	"strconv"
	"testing"
)

// TestCheckSeconds pins the last whole second that fits in a duration and the next value.
func TestCheckSeconds(t *testing.T) {
	for text, valid := range map[string]bool{"0": true, "1": true, "9223372036": true, "9223372037": false} {
		n, err := strconv.Atoi(text)
		if err != nil {
			continue
		} // Wider inputs are already rejected on 32-bit platforms.
		if err := CheckSeconds(n, "timeout_seconds"); (err == nil) != valid {
			t.Errorf("%s: %v", text, err)
		}
	}
}
