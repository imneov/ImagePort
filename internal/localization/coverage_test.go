package localization

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestApplicationErrorCoverage keeps application-owned literal errors translatable.
func TestApplicationErrorCoverage(test *testing.T) {
	var messages map[string]string
	if err := json.Unmarshal(chinese, &messages); err != nil {
		test.Fatal(err)
	}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if message := literalError(node); message != "" && messages[message] == "" {
				test.Errorf("missing Chinese error translation in %s: %s", path, message)
			}
			return true
		})
		return nil
	})
	if err != nil {
		test.Fatal(err)
	}
}

// literalError returns an errors.New string literal, or empty for unrelated AST nodes.
func literalError(node ast.Node) string {
	call, ok := node.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return ""
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "New" {
		return ""
	}
	owner, ok := selector.X.(*ast.Ident)
	if !ok || owner.Name != "errors" {
		return ""
	}
	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	message, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return message
}
