//go:build !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import "syscall"

// regularOpenFlags keeps opening a FIFO nonblocking before the file-type check.
const regularOpenFlags = syscall.O_NONBLOCK
