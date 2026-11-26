//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// diagnosticConfigReadFlags open a config for best-effort error formatting.
const diagnosticConfigReadFlags = os.O_RDONLY | unix.O_NONBLOCK

// openConfigHandle opens a config without truncating or replacing the selected inode.
var openConfigHandle = os.OpenFile

// withConfigHandleLock acquires a descriptor-owned lock without waiting for another command.
func withConfigHandleLock(file *os.File, exclusive bool, action func(*os.File) error) error {
	mode := unix.LOCK_SH | unix.LOCK_NB
	if exclusive {
		mode = unix.LOCK_EX | unix.LOCK_NB
	}
	err := unix.Flock(int(file.Fd()), mode)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return errConfigChanged
	}
	if err != nil {
		return err
	}
	return action(file)
}
