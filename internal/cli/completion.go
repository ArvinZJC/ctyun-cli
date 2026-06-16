/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func runCompletion(stdout io.Writer, args []string, installedRoot string) error {
	if len(args) != 1 {
		return fmt.Errorf("completion requires one shell: bash, zsh, or fish")
	}
	words := completionWords(installedRoot)
	switch args[0] {
	case "zsh":
		fmt.Fprintln(stdout, "#compdef ctyun")
		fmt.Fprintf(stdout, "_ctyun() { _arguments '*::ctyun command:((%s))' }\n", strings.Join(words, " "))
		return nil
	case "bash":
		fmt.Fprintf(stdout, "complete -W '%s' ctyun\n", strings.Join(words, " "))
		return nil
	case "fish":
		fmt.Fprintf(stdout, "complete -c ctyun -f -a '%s'\n", strings.Join(words, " "))
		return nil
	default:
		return fmt.Errorf("unsupported shell %q", args[0])
	}
}

func completionWords(installedRoot string) []string {
	// seen is a set; the empty struct values are placeholders for unique words.
	seen := map[string]struct{}{
		"version": {}, "upgrade": {}, "doctor": {}, "plugin": {}, "plugins": {}, "completion": {}, "help": {},
		"install": {}, "list": {}, "lint": {}, "remove": {}, "search": {}, "update": {}, "network": {},
		"--registry": {}, "--channel": {},
	}
	add := func(word string) {
		seen[word] = struct{}{}
	}
	for _, option := range globalOptionsHelp {
		add(option.Long)
		for _, alias := range option.Aliases {
			add(alias)
		}
		if option.Short != "" {
			add(option.Short)
		}
	}
	for _, bundle := range mustLoadBundlesForCompletion(installedRoot) {
		for _, command := range bundle.Commands.Commands {
			for _, part := range command.Path {
				if strings.HasPrefix(part, "{") {
					continue
				}
				add(part)
			}
			for _, parameter := range command.Parameters {
				if parameter.Flag != "" {
					add("--" + parameter.Flag)
				}
			}
		}
	}
	words := make([]string, 0, len(seen))
	for word := range seen {
		words = append(words, word)
	}
	sortStrings(words)
	return words
}

func mustLoadBundlesForCompletion(installedRoot string) []plugin.Bundle {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return nil
	}
	return bundles
}
