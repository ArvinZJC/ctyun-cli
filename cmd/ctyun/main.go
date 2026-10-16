/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"os"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// exitProcess is the process termination boundary, replaceable for entrypoint tests.
var exitProcess = os.Exit

// main runs the ctyun command with process arguments and exits with its status.
func main() {
	exitProcess(cli.Execute(cli.Config{Args: os.Args[1:]}))
}
