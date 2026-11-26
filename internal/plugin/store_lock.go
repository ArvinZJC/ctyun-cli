/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"os"

	"github.com/ArvinZJC/ctyun-cli/internal/dirlock"
)

// WithStoreLock protects a plugin inventory read or a short filesystem mutation.
// The lock uses the directory identity, so aliases share it. Missing stores have
// no installed bundles and can be inspected without creating a directory.
func WithStoreLock(root string, exclusive bool, action func() error) (err error) {
	info, err := os.Stat(root)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return action()
	}
	if err != nil {
		return err
	}
	return dirlock.With(root, exclusive, action)
}
