//go:build !darwin && !linux && !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "os"

// renameConfigNoReplace publishes a second name for the completed temp file without overwriting.
var renameConfigNoReplace = os.Link
