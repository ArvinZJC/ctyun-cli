//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"os"
	"sync"
)

// diagnosticConfigReadFlags open a config for best-effort error formatting.
const diagnosticConfigReadFlags = os.O_RDONLY

// openConfigHandle opens the selected config on platforms without descriptor locks.
var openConfigHandle = os.OpenFile

// fallbackConfigLock serializes local mutations on non-release platforms without a native lock implementation.
var fallbackConfigLock sync.RWMutex

// withConfigHandleLock provides process-local exclusion and retains snapshot checks on these platforms.
func withConfigHandleLock(file *os.File, exclusive bool, action func(*os.File) error) error {
	if exclusive {
		if !fallbackConfigLock.TryLock() {
			return errConfigChanged
		}
		defer fallbackConfigLock.Unlock()
	} else {
		if !fallbackConfigLock.TryRLock() {
			return errConfigChanged
		}
		defer fallbackConfigLock.RUnlock()
	}
	return action(file)
}
