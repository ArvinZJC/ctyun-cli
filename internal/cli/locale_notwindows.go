//go:build !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

// readWindowsUserLocale is a no-op on non-Windows platforms.
var readWindowsUserLocale = func() string {
	return ""
}
