package i18n

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLocaleAndFallback(t *testing.T) {
	defer Set("zh-CN", "")
	for _, tc := range []struct{ choice, system, want string }{
		{"", "zh-TW", "zh-CN"}, {"system", "en-US", "en"}, {"system", "fr-FR", "en"},
		{"en", "zh-CN", "en"}, {"zh-CN", "en-US", "zh-CN"},
	} {
		if got := Resolve(tc.choice, tc.system); got != tc.want {
			t.Errorf("Resolve(%q,%q)=%q", tc.choice, tc.system, got)
		}
	}
	Set("en", "")
	if T("翻译") != "Translate" || ProductName() != "squirrel_translate" {
		t.Fatal("English catalog not applied")
	}
	if got := fmt.Sprintf(T("%d 字符"), 42); got != "42 characters" {
		t.Fatal(got)
	}
	if T("unrecognized model output") != "unrecognized model output" {
		t.Fatal("Unknown strings must be preserved")
	}
	Set("zh-CN", "")
	if T("翻译") != "翻译" || ProductName() != "松鼠翻译" {
		t.Fatal("Chinese fallback changed")
	}
}

// Every explicit message reference must have a nonempty English translation.
func TestCatalogCoverage(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, e := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if e != nil {
			return e
		}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || len(c.Args) == 0 {
				return true
			}
			match := false
			switch fn := c.Fun.(type) {
			case *ast.Ident:
				match = fn.Name == "tr"
			case *ast.SelectorExpr:
				if pkg, ok := fn.X.(*ast.Ident); ok {
					match = pkg.Name == "i18n" && fn.Sel.Name == "T"
				}
			}
			if match {
				if literal, ok := c.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					key, _ := strconv.Unquote(literal.Value)
					if english[key] == "" {
						t.Errorf("%s: missing English key %q", path, key)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
