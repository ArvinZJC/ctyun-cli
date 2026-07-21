/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
)

// main runs the repository-local OpenAPI maintenance tool.
func main() {
	if err := run(os.Args[1:], ".", os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches one OpenAPI maintenance command.
func run(args []string, root string, stdout io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: openapi <harvest|diff|normalize-labels|generate|review|promote> <product>")
	}
	command := args[0]
	product := args[1]
	workspace := openapipipeline.Workspace{Root: root}
	switch command {
	case "harvest":
		fs := flag.NewFlagSet("harvest", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		input := fs.String("input", "", "normalized OpenAPI JSON input")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *input == "" {
			return fmt.Errorf("harvest requires --input")
		}
		if err := workspace.HarvestFromFile(product, *input); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "wrote %s\n", filepath.ToSlash(filepath.Join("openapi-catalogs", product, "source.json")))
		return err
	case "diff":
		if _, err := workspace.WriteDiff(product); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "wrote %s\n", filepath.ToSlash(filepath.Join("openapi-catalogs", product, "changes.md")))
		return err
	case "normalize-labels":
		changed, err := workspace.NormalizeSourceLabels(product)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "normalized %d labels in %s\n", changed, filepath.ToSlash(filepath.Join("openapi-catalogs", product, "source.json")))
		return err
	case "generate":
		if err := workspace.GenerateDraft(product); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "wrote %s\n", filepath.ToSlash(filepath.Join("openapi-catalogs", product, "draft")))
		return err
	case "review":
		report, err := workspace.ReviewDraft(product)
		if err != nil {
			return err
		}
		if !report.Ready {
			return fmt.Errorf("review blocked for %s", product)
		}
		_, err = fmt.Fprintf(stdout, "review ready for %s\n", product)
		return err
	case "promote":
		target := filepath.Join(root, "plugins", product)
		if err := workspace.PromoteDraft(product, target); err != nil {
			return err
		}
		_, err := fmt.Fprintf(stdout, "promoted %s\n", filepath.ToSlash(filepath.Join("plugins", product)))
		return err
	default:
		return fmt.Errorf("unknown command %s", command)
	}
}
