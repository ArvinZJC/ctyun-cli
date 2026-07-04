/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// confirmDangerousOperation prompts for confirmation unless --yes was used.
func confirmDangerousOperation(stderr io.Writer, stdin io.Reader, opts globalOptions, subject string) error {
	if opts.Yes {
		return nil
	}
	if _, err := fmt.Fprint(stderr, messagef("confirmation.prompt", opts.Language, subject)); err != nil {
		return err
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return diagnostic.New("error.confirmation_cancelled")
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer == "y" || answer == "yes" {
		return nil
	}
	return diagnostic.New("error.confirmation_cancelled")
}
