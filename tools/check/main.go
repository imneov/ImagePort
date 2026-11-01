// Command check enforces the repository's documented source size limits.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// main checks Go files below the current directory, excluding Git and generated data.
func main() {
	failed := false
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "tmp" || entry.Name() == "dist") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		issues, err := check(path)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			fmt.Fprintln(os.Stderr, issue)
			failed = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}

// check reports overlong files/functions and undocumented named functions in path.
func check(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, path, data, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var issues []string
	if strings.Count(string(data), "\n") > 1000 {
		issues = append(issues, path+": exceeds 1000 lines")
	}
	ast.Inspect(file, func(node ast.Node) bool {
		var start, end token.Pos
		switch function := node.(type) {
		case *ast.FuncDecl:
			start, end = function.Pos(), function.End()
			if function.Doc == nil {
				issues = append(issues, positions.Position(start).String()+": missing function documentation")
			}
		case *ast.FuncLit:
			start, end = function.Pos(), function.End()
		default:
			return true
		}
		lines := positions.Position(end).Line - positions.Position(start).Line + 1
		if lines > 50 {
			issues = append(issues, fmt.Sprintf("%s: function exceeds 50 lines (%d)", positions.Position(start), lines))
		}
		return true
	})
	return issues, nil
}
