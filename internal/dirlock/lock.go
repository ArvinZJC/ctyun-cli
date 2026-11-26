/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package dirlock coordinates access to an existing directory across cooperating processes.
package dirlock

import (
	"errors"
	"os"
)

// With runs action while holding a lock on the opened directory. Directory aliases
// share the lock. Closing the descriptor releases its resources on every return path.
func With(root string, exclusive bool, action func() error) (err error) {
	dir, err := os.Open(root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, dir.Close()) }()
	return lockDirectory(dir, exclusive, action)
}
