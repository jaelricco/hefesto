package domain_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbidden lists the packages internal/domain must never depend on.
//
// The rule from the project conventions: internal/domain is pure types and
// business rules with zero I/O. It may not reach into persistence or
// transport. This test is the enforcement, so the rule cannot rot silently.
var forbidden = []string{
	"github.com/jaelricco/hefesto/internal/store",
	"github.com/jaelricco/hefesto/internal/http",
	"github.com/jaelricco/hefesto/internal/auth",
	"github.com/jaelricco/hefesto/internal/media",
	"github.com/jaelricco/hefesto/internal/sync",
	"database/sql",
	"net/http",
	"os",
}

func TestDomainHasNoInfrastructureImports(t *testing.T) {
	root := "."
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			p, uerr := strconv.Unquote(imp.Path.Value)
			if uerr != nil {
				continue
			}
			for _, bad := range forbidden {
				if p == bad || strings.HasPrefix(p, bad+"/") {
					t.Errorf("%s imports %q — internal/domain must stay free of I/O and infrastructure", path, p)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking domain tree: %v", err)
	}
}
