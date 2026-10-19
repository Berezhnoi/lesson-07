package web_test

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/net/html"

	webui "homework/web"
)

// unsafeTemplateTypeAllowlist is intentionally empty. Any future exception must
// name the exact source file and explain why escaping can safely be bypassed.
var unsafeTemplateTypeAllowlist = map[string]string{}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate audit_test.go")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
}

func TestGoSourcesAvoidUnsafeTemplateAPIs(t *testing.T) {
	root := repositoryRoot(t)
	fset := token.NewFileSet()
	unsafeTypes := map[string]bool{
		"HTML": true, "JS": true, "CSS": true, "URL": true,
		"HTMLAttr": true, "JSStr": true, "Srcset": true,
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, imported := range file.Imports {
			if imported.Path.Value == `"text/template"` {
				t.Errorf("%s imports text/template; use html/template", path)
			}
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if reason, allowed := unsafeTemplateTypeAllowlist[filepath.ToSlash(relative)]; allowed {
			if strings.TrimSpace(reason) == "" {
				t.Errorf("unsafe template allowlist entry %q needs a reason", relative)
			}
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil || !unsafeTypes[selector.Sel.Name] {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "template" {
				t.Errorf("%s uses template.%s; keep untrusted values escaped", relative, selector.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedTemplatesAreSafeAndExecuteWithoutWorkingFiles(t *testing.T) {
	_ = repositoryRoot(t) // Fail early if the test is run from an unexpected source layout.
	templates := make([]string, 0)
	err := fs.WalkDir(webui.TemplateFS, "templates", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".html") {
			templates = append(templates, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("no embedded HTML templates found")
	}

	for _, path := range templates {
		t.Run(path, func(t *testing.T) {
			source, err := fs.ReadFile(webui.TemplateFS, path)
			if err != nil {
				t.Fatal(err)
			}
			assertSafeTemplateMarkup(t, path, string(source))

			parsed, err := template.New(filepath.Base(path)).ParseFS(webui.TemplateFS, path)
			if err != nil {
				t.Fatalf("parse embedded template: %v", err)
			}
			// A blank directory proves parsing and execution use only the embedded FS.
			emptyDir := t.TempDir()
			previous, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(emptyDir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Chdir(previous); err != nil {
					t.Errorf("restore working directory: %v", err)
				}
			})
			if err := parsed.ExecuteTemplate(ioDiscard{}, filepath.Base(path), map[string]any{}); err != nil {
				t.Errorf("execute embedded template from empty working directory: %v", err)
			}
		})
	}

	static, err := webui.StaticFS()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"css/style.css", "js/htmx.min.js", "js/app.js", "js/VENDORED.txt"} {
		if _, err := fs.Stat(static, required); err != nil {
			t.Errorf("embedded static file %q is missing: %v", required, err)
		}
	}
	js, err := fs.ReadFile(static, "js/htmx.min.js")
	if err != nil {
		t.Fatal(err)
	}
	vendored, err := fs.ReadFile(static, "js/VENDORED.txt")
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(js))
	if !strings.Contains(string(vendored), "SHA-256: "+digest) {
		t.Errorf("VENDORED.txt does not record SHA-256 %s", digest)
	}

}

// ioDiscard is a tiny writer so template execution does not retain output.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

var unescapingPipeline = regexp.MustCompile(`(?i)\|\s*(safe(html|js|css|url|htmlattr|jsstr|srcset)?|html|unescape(d)?|trust(html|js|css)?|raw|tohtml|tojs|template\.(html|js|css|url|htmlattr|jsstr|srcset))\b`)

func assertSafeTemplateMarkup(t *testing.T, name, source string) {
	t.Helper()
	if match := unescapingPipeline.FindString(source); match != "" {
		t.Errorf("%s has an unescaping template pipeline %q", name, match)
	}
	root, err := html.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatalf("parse template markup: %v", err)
	}
	var inspect func(*html.Node)
	inspect = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attribute := range node.Attr {
				key := strings.ToLower(attribute.Key)
				value := strings.ToLower(strings.TrimSpace(attribute.Val))
				if strings.HasPrefix(key, "on") && len(key) > 2 {
					t.Errorf("%s has inline event handler %s=", name, key)
				}
				if strings.HasPrefix(value, "javascript:") {
					t.Errorf("%s has javascript: URL in %s", name, key)
				}
			}
			if node.Data == "script" {
				hasSource := false
				for _, attribute := range node.Attr {
					if strings.EqualFold(attribute.Key, "src") {
						hasSource = true
					}
				}
				if !hasSource && strings.TrimSpace(nodeText(node)) != "" {
					t.Errorf("%s has an inline script body", name)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			inspect(child)
		}
	}
	inspect(root)
}

func nodeText(node *html.Node) string {
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if current.Type == html.TextNode {
			text.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return text.String()
}

var _ fs.FS = webui.TemplateFS
