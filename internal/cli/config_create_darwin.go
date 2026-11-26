/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "golang.org/x/sys/unix"

// renameConfigNoReplace publishes a completed file only if its destination is absent.
func renameConfigNoReplace(from, to string) error {
	return unix.RenamexNp(from, to, unix.RENAME_EXCL)
}
