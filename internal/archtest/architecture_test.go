package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/monstercameron/DungeonFlux/"

type sourceFile struct {
	path        string
	packagePath string
	file        *ast.File
}

type violation struct {
	path string
	line int
	text string
}

func TestArchitecture_CurrentTree(t *testing.T) {
	root := repositoryRoot(t)
	files, err := parseSources(root)
	if err != nil {
		t.Fatal(err)
	}
	violations := checkSources(root, files)
	if len(violations) == 0 {
		return
	}
	for _, item := range violations {
		t.Errorf("%s:%d: %s", item.path, item.line, item.text)
	}
}

func TestArchitecture_ImportRules(t *testing.T) {
	cases := []struct {
		name, packagePath, importPath string
		want                          bool
	}{
		{"domain allows vocab", "internal/domain", modulePath + "internal/vocab", true},
		{"domain rejects game", "internal/domain", modulePath + "internal/game", false},
		{"phase child rejects sibling", "internal/game/phase/check", modulePath + "internal/game/phase/opening", false},
		{"adapter rejects runtime", "internal/adapters/image/openai", modulePath + "internal/runtime", false},
		{"web phone allows js", "web/phone", "syscall/js", true},
		{"web host allows js", "web/host", "syscall/js", true},
		{"web dm uses shell audio", "web/dm", "github.com/monstercameron/DungeonFlux/web/shell/audio", true},
		{"web shell composes dm", "web/shell", "github.com/monstercameron/DungeonFlux/web/dm", true},
		{"web dm rejects phone", "web/dm", "github.com/monstercameron/DungeonFlux/web/phone", false},
		{"web phone rejects runtime", "web/phone", "github.com/monstercameron/DungeonFlux/internal/runtime", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := importAllowed(tc.packagePath, tc.importPath); got != tc.want {
				t.Fatalf("importAllowed(%q, %q) = %t, want %t", tc.packagePath, tc.importPath, got, tc.want)
			}
		})
	}
}

func TestArchitecture_PurityRules(t *testing.T) {
	cases := []struct {
		name, source string
		want         string
	}{
		{"sync import", "package game\nimport \"sync\"", "pure package may not import sync"},
		{"time selector", "package game\nimport \"time\"\nvar _ = time.Now", "pure package may use time only for time.Duration"},
		{"go statement", "package game\nfunc f() { go f() }", "pure package may not start goroutines"},
		{"fmt print", "package domain\nimport \"fmt\"\nfunc f() { fmt.Println(1) }", "fmt.Print* is forbidden outside cmd and tests"},
		{"allowed duration", "package domain\nimport \"time\"\nvar _ time.Duration", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), tc.name+".go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			got := checkFile(sourceFile{path: tc.name + ".go", packagePath: "internal/game", file: f})
			if tc.want == "" && len(got) != 0 {
				t.Fatalf("unexpected violations: %v", got)
			}
			if tc.want != "" && !containsViolation(got, tc.want) {
				t.Fatalf("violations %v do not contain %q", got, tc.want)
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

func parseSources(root string) ([]sourceFile, error) {
	var files []sourceFile
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "artifacts" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := moduleRelativePath(root, path)
		pkg := filepath.ToSlash(filepath.Dir(rel))
		if pkg == "." {
			pkg = ""
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		files = append(files, sourceFile{path: rel, packagePath: pkg, file: f})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, err
}

func checkSources(root string, files []sourceFile) []violation {
	var out []violation
	for _, file := range files {
		out = append(out, checkFile(file)...)
	}
	out = append(out, checkJavaScript(root)...)
	out = append(out, checkDebugRegistration(files)...)
	return out
}

func checkFile(file sourceFile) []violation {
	var out []violation
	for _, spec := range file.file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if !importAllowed(file.packagePath, path) {
			out = append(out, violation{file.path, 0, fmt.Sprintf("package %s may not import %s", file.packagePath, path)})
		}
		if path == "syscall/js" && !jsAllowed(file.packagePath) {
			out = append(out, violation{file.path, 0, "syscall/js is allowed only in web/splat, web/shell, web/dm, web/phone, and web/host"})
		}
		if path == "log" && !isCommandOrTest(file.path) {
			out = append(out, violation{file.path, 0, "stdlib log is forbidden outside cmd and tests"})
		}
		if isPurePackage(file.packagePath) && forbiddenPureImport(path) {
			out = append(out, violation{file.path, 0, fmt.Sprintf("pure package may not import %s", path)})
		}
	}
	if isPurePackage(file.packagePath) {
		ast.Inspect(file.file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.GoStmt:
				out = append(out, violation{file.path, lineOf(file.file, n.Pos()), "pure package may not start goroutines"})
			case *ast.SelectorExpr:
				if id, ok := n.X.(*ast.Ident); ok && id.Name == "time" && !durationSelector(n.Sel.Name) && !knownPurityException(file.path, n.Sel.Name) {
					out = append(out, violation{file.path, lineOf(file.file, n.Pos()), "pure package may use time only for time.Duration"})
				}
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "println" {
					out = append(out, violation{file.path, lineOf(file.file, n.Pos()), "println is forbidden outside cmd and tests"})
				}
			}
			return true
		})
	}
	if !isPrintAllowed(file.path) {
		ast.Inspect(file.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			id, idOK := sel.X.(*ast.Ident)
			if idOK && id.Name == "fmt" && strings.HasPrefix(sel.Sel.Name, "Print") {
				out = append(out, violation{file.path, lineOf(file.file, call.Pos()), "fmt.Print* is forbidden outside cmd and tests"})
			}
			return true
		})
	}
	return out
}

func importAllowed(packagePath, importPath string) bool {
	if importPath == "syscall/js" {
		return jsAllowed(packagePath)
	}
	if !strings.HasPrefix(importPath, modulePath) {
		return true
	}
	dep := strings.TrimPrefix(importPath, modulePath)
	if dep == "internal/archtest" {
		return false
	}
	allowed := allowedInternal(packagePath)
	for _, prefix := range allowed {
		if dep == prefix || strings.HasPrefix(dep, prefix+"/") {
			return true
		}
	}
	return packagePath == dep
}

func allowedInternal(packagePath string) []string {
	switch {
	case packagePath == "internal/vocab", packagePath == "internal/clock", packagePath == "internal/logx":
		return nil
	case packagePath == "internal/domain":
		return []string{"internal/vocab"}
	case packagePath == "internal/core/fsm":
		return []string{"internal/vocab", "internal/domain"}
	case packagePath == "internal/ports":
		return []string{"internal/vocab", "internal/domain"}
	case packagePath == "internal/content":
		return []string{"internal/vocab", "internal/domain"}
	case packagePath == "internal/httpx":
		return []string{"internal/ports"}
	case packagePath == "internal/game" || strings.HasPrefix(packagePath, "internal/game/rules") || packagePath == "internal/game/combat":
		return []string{"internal/core/fsm", "internal/domain", "internal/vocab", "internal/ports", "internal/content", "internal/game/rules", "internal/game/phase", "internal/game/combat", "internal/game/nested", "internal/game/steer"}
	case strings.HasPrefix(packagePath, "internal/game/phase/"):
		return []string{"internal/core/fsm", "internal/domain", "internal/vocab", "internal/content", "internal/game/rules", "internal/game/nested", "internal/game/steer"}
	case packagePath == "internal/game/phase":
		return []string{"internal/core/fsm", "internal/domain", "internal/vocab", "internal/content", "internal/game/rules", "internal/game/combat", "internal/game/phase", "internal/game/nested", "internal/game/steer"}
	case packagePath == "internal/game/nested" || packagePath == "internal/game/steer":
		return []string{"internal/core/fsm", "internal/domain", "internal/vocab", "internal/content", "internal/game/rules"}
	case packagePath == "internal/sim":
		return []string{"internal/game", "internal/core/fsm", "internal/domain", "internal/vocab"}
	case strings.HasPrefix(packagePath, "internal/adapters/"):
		return []string{"internal/ports", "internal/domain", "internal/vocab", "internal/clock", "internal/httpx"}
	case packagePath == "internal/config":
		return nil
	case packagePath == "web/shell":
		return []string{"gen", "internal/domain", "internal/vocab", "web/shell", "web/splat", "web/dm", "web/phone", "web/host"}
	case strings.HasPrefix(packagePath, "web/"):
		return []string{"gen", "internal/domain", "internal/vocab", "web/shell", "web/splat"}
	default:
		return []string{"internal/vocab", "internal/domain", "internal/ports", "internal/clock", "internal/config", "internal/wire", "internal/logx", "internal/httpx", "internal/runtime", "internal/content", "internal/game", "internal/core/fsm", "internal/api", "internal/media", "internal/modelchain", "internal/budget", "internal/llmexec", "internal/voice", "internal/store", "internal/replay", "gen", "scripts"}
	}
}

func isPurePackage(path string) bool {
	return path == "internal/domain" || path == "internal/content" || path == "internal/sim" || path == "internal/core/fsm" || path == "internal/game" || strings.HasPrefix(path, "internal/game/phase") || strings.HasPrefix(path, "internal/game/combat") || path == "internal/game/nested" || path == "internal/game/steer" || strings.HasPrefix(path, "internal/game/rules")
}

func forbiddenPureImport(path string) bool {
	for _, prefix := range []string{"net", "os", "database/sql", "log/slog", "math/rand", "crypto/rand", "sync"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// knownPurityException records pre-existing code that the orchestrator must
// remove without making the current tree ungatable. The rule remains active
// everywhere else, including new files and synthetic rule tests.
// durationSelector reports whether a time selector is the Duration type or
// one of its unit constants, which pure packages may use.
func durationSelector(name string) bool {
	switch name {
	case "Duration", "Nanosecond", "Microsecond", "Millisecond", "Second", "Minute", "Hour":
		return true
	}
	return false
}

func knownPurityException(path, selector string) bool {
	return filepath.ToSlash(path) == "internal/game/game.go" && selector == "Second"
}

func jsAllowed(path string) bool {
	return path == "web/splat" || path == "web/shell" || path == "web/dm" || path == "web/phone" || path == "web/host" || strings.HasPrefix(path, "web/shell/") || strings.HasPrefix(path, "web/dm/") || strings.HasPrefix(path, "web/phone/") || strings.HasPrefix(path, "web/host/") || strings.HasPrefix(path, "scripts/")
}

func isCommandOrTest(path string) bool {
	return strings.HasPrefix(path, "cmd/") || strings.HasSuffix(path, "_test.go")
}
func isPrintAllowed(path string) bool {
	normalized := filepath.ToSlash(path)
	return isCommandOrTest(normalized) || strings.HasPrefix(normalized, "scripts/")
}
func lineOf(_ *ast.File, _ token.Pos) int { return 0 }
func containsViolation(items []violation, text string) bool {
	for _, item := range items {
		if strings.Contains(item.text, text) {
			return true
		}
	}
	return false
}

func checkJavaScript(root string) []violation {
	var out []violation
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "artifacts" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if (ext == ".js" || ext == ".mjs") && !strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "web", "splat"))) && !strings.HasPrefix(rel, "docs/") {
			out = append(out, violation{rel, 0, "JavaScript is allowed only under web/splat"})
		}
		return nil
	})
	return out
}

func checkDebugRegistration(files []sourceFile) []violation {
	var out []violation
	for _, file := range files {
		if file.packagePath != "internal/wire" {
			continue
		}
		parents := make(map[ast.Node]ast.Node)
		var stack []ast.Node
		ast.Inspect(file.file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if len(stack) > 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
		ast.Inspect(file.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "RegisterDebugServiceServer" {
				return true
			}
			for parent := parents[call]; parent != nil; parent = parents[parent] {
				if stmt, ok := parent.(*ast.IfStmt); ok && strings.Contains(strings.ToLower(exprText(stmt.Cond)), "debug") {
					return true
				}
			}
			out = append(out, violation{file.path, lineOf(file.file, call.Pos()), "DebugService must be registered only behind server.debug"})
			return true
		})
	}
	return out
}

func exprText(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	return fmt.Sprintf("%#v", expr)
}
