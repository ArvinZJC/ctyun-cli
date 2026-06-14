/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"os"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.Config{Args: os.Args[1:]}))
}
