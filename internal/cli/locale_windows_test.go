//go:build windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "testing"

func TestReadWindowsUserLocale(t *testing.T) {
	_ = readWindowsUserLocale()
}
