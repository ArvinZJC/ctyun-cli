//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package dirlock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// flock is the descriptor-locking boundary used by interruption and failure tests.
var flock = unix.Flock

// lockDirectory allows parallel readers and serializes directory publication and removal.
func lockDirectory(dir *os.File, exclusive bool, action func() error) error {
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	for {
		err := flock(int(dir.Fd()), mode)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		return action()
	}
}
