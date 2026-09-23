/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package coverprofile

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// TestPartialExclusionsStillIdentifySourceBlocks catches removed or shifted
// exclusions before stale line numbers silently weaken the coverage policy.
func TestPartialExclusionsStillIdentifySourceBlocks(t *testing.T) {
	for _, exclusion := range DefaultExclusions() {
		if exclusion.WholeFile {
			continue
		}
		t.Run(exclusion.File+":"+strconv.Itoa(exclusion.StartLine), func(t *testing.T) {
			positions := token.NewFileSet()
			source, err := parser.ParseFile(positions, filepath.Join("..", "..", exclusion.File), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			matched := false
			ast.Inspect(source, func(node ast.Node) bool {
				block, ok := node.(*ast.BlockStmt)
				if !ok || positions.Position(block.Lbrace).Line != exclusion.StartLine {
					return true
				}
				if positions.Position(block.Rbrace).Line == exclusion.EndLine {
					matched = true
				}
				// Coverage splits a return expression before an inline cleanup
				// function, so its block ends at that function's opening line.
				ast.Inspect(block, func(child ast.Node) bool {
					literal, ok := child.(*ast.FuncLit)
					if ok && positions.Position(literal.Body.Lbrace).Line == exclusion.EndLine {
						matched = true
					}
					return true
				})
				return true
			})
			if !matched {
				t.Fatalf("stale exclusion %s:%d-%d does not match a source block", exclusion.File, exclusion.StartLine, exclusion.EndLine)
			}
		})
	}
}
