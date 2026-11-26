//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package dirlock

import (
	"os"
	"sync"
)

// localLock protects concurrent callers on platforms without a release binary.
var localLock sync.RWMutex

// lockDirectory serializes mutations within the process on these platforms.
func lockDirectory(_ *os.File, exclusive bool, action func() error) error {
	if exclusive {
		localLock.Lock()
		defer localLock.Unlock()
	} else {
		localLock.RLock()
		defer localLock.RUnlock()
	}
	return action()
}
