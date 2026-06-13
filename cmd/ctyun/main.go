package main

import (
	"os"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.Config{Args: os.Args[1:]}))
}
