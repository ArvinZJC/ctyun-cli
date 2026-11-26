//go:build !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"fmt"
	"os"
	"time"
)

// publishExecutable unlinks the previous executable after a successful replacement.
func publishExecutable(current, staged string, renamePath func(string, string) error) error {
	backup := current + ".old-" + fmt.Sprint(time.Now().UnixNano())
	if err := replaceWithBackup(current, staged, backup, renamePath); err != nil {
		return err
	}
	return os.Remove(backup)
}
