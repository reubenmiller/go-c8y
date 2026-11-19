package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// httpMethods maps the selector names used for HTTP methods (resty.MethodGet,
// http.MethodGet) to the OAS method.
var httpMethods = map[string]string{
	"MethodGet":    "GET",
	"MethodPost":   "POST",
	"MethodPut":    "PUT",
	"MethodDelete": "DELETE",
	"MethodPatch":  "PATCH",
	"MethodHead":   "HEAD",
}

// scanSDKMethods infers, per normalised path, the HTTP methods the SDK sends to
// it. In each function it pairs the SetMethod(...) argument with every
// SetURL(...) argument; URL arguments are resolved through package-level
// string constants and variables. Functions with zero or several distinct
// methods are skipped, so their paths count as "methods unknown".
func scanSDKMethods(srcDir string) (map[string]map[string]bool, error) {
	fset := token.NewFileSet()
	// Package name -> identifier -> string value, built from all non-test files.
	idents := map[string]map[string]string{}
	type fileInfo struct {
		pkg  string
		file *ast.File
		gen  bool
	}
	var files []fileInfo

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		pkg := f.Name.Name
		if idents[pkg] == nil {
			idents[pkg] = map[string]string{}
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || (gd.Tok != token.CONST && gd.Tok != token.VAR) {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if v, err := strconv.Unquote(lit.Value); err == nil {
							idents[pkg][name.Name] = v
						}
					}
				}
			}
		}
		files = append(files, fileInfo{pkg: pkg, file: f, gen: strings.HasPrefix(d.Name(), "zz_generated")})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// SetURL arguments are known to be URLs, so any absolute path counts
	// (looksLikeAPIPath would drop single-segment paths such as /features/{key}).
	isURLPath := func(v string) bool {
		return strings.HasPrefix(v, "/") && len(v) > 1 && !strings.ContainsAny(v, " *")
	}
	resolve := func(pkg string, e ast.Expr) []string {
		var out []string
		var walk func(ast.Expr)
		walk = func(e ast.Expr) {
			switch x := e.(type) {
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					if v, err := strconv.Unquote(x.Value); err == nil && isURLPath(v) {
						out = append(out, v)
					}
				}
			case *ast.Ident:
				if v, ok := idents[pkg][x.Name]; ok && isURLPath(v) {
					out = append(out, v)
				}
			case *ast.SelectorExpr:
				if p, ok := x.X.(*ast.Ident); ok {
					if v, ok := idents[p.Name][x.Sel.Name]; ok && isURLPath(v) {
						out = append(out, v)
					}
				}
			case *ast.CallExpr:
				for _, a := range x.Args {
					walk(a)
				}
			}
		}
		walk(e)
		return out
	}

	methodOf := func(e ast.Expr) string {
		switch x := e.(type) {
		case *ast.SelectorExpr:
			return httpMethods[x.Sel.Name]
		case *ast.BasicLit:
			if v, err := strconv.Unquote(x.Value); err == nil {
				return strings.ToUpper(v)
			}
		}
		return ""
	}

	out := map[string]map[string]bool{}
	for _, fi := range files {
		if fi.gen {
			continue
		}
		for _, decl := range fi.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			methods := map[string]bool{}
			var urls []string
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch sel.Sel.Name {
				case "SetMethod":
					if m := methodOf(call.Args[0]); m != "" {
						methods[m] = true
					}
				case "SetURL":
					urls = append(urls, resolve(fi.pkg, call.Args[0])...)
				}
				return true
			})
			if len(methods) != 1 || len(urls) == 0 {
				continue
			}
			for m := range methods {
				for _, u := range urls {
					np := normalizePath(u)
					if out[np] == nil {
						out[np] = map[string]bool{}
					}
					out[np][m] = true
				}
			}
		}
	}
	return out, nil
}

// splitWaiver splits a waiver into an optional HTTP method and a path pattern:
// "POST /a/b" -> ("POST", "/a/b"); "/a/b" -> ("", "/a/b").
func splitWaiver(w string) (method, pattern string) {
	if m, p, ok := strings.Cut(strings.TrimSpace(w), " "); ok {
		if _, known := httpMethods["Method"+strings.ToUpper(m[:1])+strings.ToLower(m[1:])]; known {
			return strings.ToUpper(m), strings.TrimSpace(p)
		}
	}
	return "", strings.TrimSpace(w)
}

type opWaiver struct {
	method  string
	pattern string
}

func parseOpWaivers(waivers []string) []opWaiver {
	out := make([]opWaiver, 0, len(waivers))
	for _, w := range waivers {
		m, p := splitWaiver(w)
		out = append(out, opWaiver{method: m, pattern: normalizePatterns([]string{p})[0]})
	}
	return out
}

func opWaived(method, normPath string, waivers []opWaiver) bool {
	for _, w := range waivers {
		if w.method != "" && w.method != method {
			continue
		}
		if matchesAny(normPath, []string{w.pattern}) {
			return true
		}
	}
	return false
}

// CheckWaiverReasons reports drift waivers in the overlay that have no
// reason: neither an inline "# ..." comment nor a comment line heading their
// group.
func CheckWaiverReasons(overlayPath string) ([]string, error) {
	f, err := os.Open(overlayPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var missing []string
	inDrift := false
	groupHasComment := false
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(line, " ") && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			inDrift = strings.HasPrefix(trimmed, "drift:")
			groupHasComment = false
			continue
		}
		if !inDrift {
			continue
		}
		switch {
		case trimmed == "":
			groupHasComment = false
		case strings.HasPrefix(trimmed, "#"):
			groupHasComment = true
		case strings.HasPrefix(trimmed, "- "):
			if !strings.Contains(trimmed, " #") && !groupHasComment {
				missing = append(missing, fmt.Sprintf("line %d: %s", lineNo, trimmed))
			}
		default:
			// A key such as "ignoreMissing:" starts a new group.
			groupHasComment = false
		}
	}
	return missing, scanner.Err()
}

// opString formats an operation for reports.
func opString(method, path string) string {
	return method + " " + path
}
