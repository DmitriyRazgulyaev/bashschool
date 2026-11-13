package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// These tests protect architectural boundaries, not individual implementations.
func TestLayerDependencies(t *testing.T) {
	sql := regexp.MustCompile(`(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|WITH|CREATE TABLE|ALTER TABLE)\b`)
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			dependency, _ := strconv.Unquote(spec.Path.Value)
			isDomain := strings.Contains(path, "/domain/")
			isService := strings.Contains(path, "/service/")
			isTransport := strings.Contains(path, "/transport/")
			forbidden := isDomain && (strings.Contains(dependency, "bashkir-school/backend/internal/") || strings.Contains(dependency, "github.com/"))
			forbidden = forbidden || isService && (strings.Contains(dependency, "/repository/") || strings.Contains(dependency, "/transport/") || strings.Contains(dependency, "gin-gonic") || strings.Contains(dependency, "pgx") || dependency == "net/http")
			forbidden = forbidden || isTransport && (strings.Contains(dependency, "/repository/") || strings.Contains(dependency, "pgx"))
			if forbidden {
				t.Errorf("%s imports forbidden dependency %s", path, dependency)
			}
		}
		if !strings.Contains(path, "/repository/postgres/") {
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(literal.Value)
				if err == nil && sql.MatchString(value) {
					t.Errorf("SQL outside PostgreSQL adapter: %s", path)
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
