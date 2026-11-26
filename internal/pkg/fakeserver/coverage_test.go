package fakeserver

// Coverage matrix: which REST endpoints the SDK's typed services call, and
// whether the fake server routes them.
//
// The endpoints are derived from the SDK source: every request builder of
// pkg/c8y/api sets its method and URL with SetMethod(...) and SetURL(...), so
// the test parses those calls with go/ast and resolves the arguments (string
// constants and vars of the same or an imported package, concatenations, and
// the arguments of helper builders such as linkB(method, path, ...), which are
// resolved at their call sites). This is more reliable than the OpenAPI spec
// files, which list endpoints the SDK does not call and miss the ones outside
// of the Cumulocity core spec (data hub, data broker, digital twin manager,
// dataprep, microservice loggers).
//
// An endpoint is "handled" when the fake server answers it with anything but
// the router's default 404 HTML page, a 405 (method not allowed), or the 404 a
// handler gives to an endpoint it does not implement (writeUnrouted, marked
// with the X-Fakeserver-Unrouted header). The 404 of a resource that does not
// exist is a handled endpoint.
//
// The committed allowlist (testdata/fakeserver_coverage_gaps.txt) lists the
// known gaps. It is a ratchet: the test fails when an endpoint is not handled
// and not in the allowlist (a new SDK endpoint), and when an endpoint in the
// allowlist is handled (remove it from the list). Regenerate the list and
// docs/FAKESERVER_COVERAGE.md with
//
//	UPDATE=1 go test ./internal/pkg/fakeserver -run TestFakeServerCoverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	coverageAPIRoot  = "../../../pkg/c8y/api"
	coverageGapsFile = "testdata/fakeserver_coverage_gaps.txt"
	coverageDocFile  = "../../../docs/FAKESERVER_COVERAGE.md"
	coverageModule   = "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
)

// sdkEndpoint is one request the SDK sends.
type sdkEndpoint struct {
	Service string // package directory below pkg/c8y/api
	Method  string
	Path    string
}

func (e sdkEndpoint) key() string { return e.Method + " " + e.Path }

type coveragePkg struct {
	dir    string
	files  []*coverageFile
	decls  map[string]ast.Expr // package level string const and var initialisers
	funcs  map[string][]*ast.FuncDecl
	fileOf map[ast.Node]*coverageFile
}

type coverageFile struct {
	file    *ast.File
	imports map[string]string // local name -> import path
	pkg     *coveragePkg
}

type coverageScanner struct {
	fset *token.FileSet
	pkgs map[string]*coveragePkg // key: directory relative to coverageAPIRoot ("." for the root)
}

func newCoverageScanner(t *testing.T) *coverageScanner {
	t.Helper()
	sc := &coverageScanner{fset: token.NewFileSet(), pkgs: map[string]*coveragePkg{}}
	err := filepath.Walk(coverageAPIRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(sc.fset, p, nil, 0)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(coverageAPIRoot, filepath.Dir(p))
		rel = filepath.ToSlash(rel)
		pkg := sc.pkgs[rel]
		if pkg == nil {
			pkg = &coveragePkg{dir: rel, decls: map[string]ast.Expr{}, funcs: map[string][]*ast.FuncDecl{}, fileOf: map[ast.Node]*coverageFile{}}
			sc.pkgs[rel] = pkg
		}
		cf := &coverageFile{file: f, imports: map[string]string{}, pkg: pkg}
		for _, imp := range f.Imports {
			ip, _ := strconv.Unquote(imp.Path.Value)
			name := filepath.Base(ip)
			if imp.Name != nil {
				name = imp.Name.Name
			}
			cf.imports[name] = ip
		}
		pkg.files = append(pkg.files, cf)
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				if d.Tok != token.CONST && d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					vs := s.(*ast.ValueSpec)
					for i, n := range vs.Names {
						if i < len(vs.Values) {
							pkg.decls[n.Name] = vs.Values[i]
							pkg.fileOf[vs.Values[i]] = cf
						}
					}
				}
			case *ast.FuncDecl:
				pkg.funcs[d.Name.Name] = append(pkg.funcs[d.Name.Name], d)
				pkg.fileOf[d] = cf
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// resolve evaluates a string (or resty.MethodX) expression. env holds the
// values of the parameters of the enclosing function.
func (sc *coverageScanner) resolve(e ast.Expr, cf *coverageFile, env map[string]string, depth int) (string, bool) {
	if depth > 8 {
		return "", false
	}
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			s, err := strconv.Unquote(e.Value)
			return s, err == nil
		}
	case *ast.ParenExpr:
		return sc.resolve(e.X, cf, env, depth+1)
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, ok1 := sc.resolve(e.X, cf, env, depth+1)
		r, ok2 := sc.resolve(e.Y, cf, env, depth+1)
		return l + r, ok1 && ok2
	case *ast.Ident:
		if v, ok := env[e.Name]; ok {
			return v, true
		}
		if d, ok := cf.pkg.decls[e.Name]; ok {
			return sc.resolve(d, cf.pkg.fileOf[d], nil, depth+1)
		}
	case *ast.SelectorExpr:
		x, ok := e.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		// net/http and resty method constants
		if strings.HasPrefix(e.Sel.Name, "Method") && (x.Name == "http" || x.Name == "resty") {
			return strings.ToUpper(strings.TrimPrefix(e.Sel.Name, "Method")), true
		}
		ip, ok := cf.imports[x.Name]
		if !ok || !strings.HasPrefix(ip, coverageModule) {
			return "", false
		}
		dir := strings.TrimPrefix(strings.TrimPrefix(ip, coverageModule), "/")
		if dir == "" {
			dir = "."
		}
		other := sc.pkgs[dir]
		if other == nil {
			return "", false
		}
		if d, ok := other.decls[e.Sel.Name]; ok {
			return sc.resolve(d, other.fileOf[d], nil, depth+1)
		}
	}
	return "", false
}

// chainSetters returns the SetMethod and SetURL argument expressions of the
// method chain call belongs to (its receivers).
func chainSetters(call *ast.CallExpr) (method, url ast.Expr) {
	var cur ast.Expr = call
	for {
		c, ok := cur.(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		switch sel.Sel.Name {
		case "SetMethod":
			if len(c.Args) == 1 && method == nil {
				method = c.Args[0]
			}
		case "SetURL":
			if len(c.Args) == 1 && url == nil {
				url = c.Args[0]
			}
		}
		cur = sel.X
	}
}

type coverageScan struct {
	endpoints  []sdkEndpoint
	unresolved []string
}

func (sc *coverageScanner) scan() coverageScan {
	var out coverageScan
	seen := map[string]bool{}
	add := func(service, method, path string) {
		method = strings.ToUpper(method)
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
		if !strings.HasPrefix(path, "/") {
			return
		}
		e := sdkEndpoint{Service: service, Method: method, Path: path}
		k := e.Service + "|" + e.key()
		if !seen[k] {
			seen[k] = true
			out.endpoints = append(out.endpoints, e)
		}
	}
	dirs := make([]string, 0, len(sc.pkgs))
	for d := range sc.pkgs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		pkg := sc.pkgs[dir]
		service := dir
		if service == "." {
			service = "client"
		}
		var names []string
		for n := range pkg.funcs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, name := range names {
			for _, fd := range pkg.funcs[name] {
				sc.scanFunc(pkg, fd, service, add, &out)
			}
		}
	}
	// request builders whose URL is computed at runtime
	var unresolved []string
	for _, u := range out.unresolved {
		switch u {
		case "client: SendRequest (unresolved)":
			// generic request escape hatch: the caller picks the path
		case "devices: createAccessTokenB (unresolved)":
			// the URL is the mTLS host (mtlsEndpoint) plus the path
			add("devices", http.MethodPost, devicesAccessTokenPath)
		default:
			unresolved = append(unresolved, u)
		}
	}
	out.unresolved = unresolved
	sort.Slice(out.endpoints, func(i, j int) bool {
		a, b := out.endpoints[i], out.endpoints[j]
		if a.Service != b.Service {
			return a.Service < b.Service
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	sort.Strings(out.unresolved)
	return out
}

func paramNames(fd *ast.FuncDecl) []string {
	var names []string
	for _, f := range fd.Type.Params.List {
		if len(f.Names) == 0 {
			names = append(names, "_")
		}
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
	}
	return names
}

func (sc *coverageScanner) scanFunc(pkg *coveragePkg, fd *ast.FuncDecl, service string, add func(service, method, path string), out *coverageScan) {
	if fd.Body == nil {
		return
	}
	cf := pkg.fileOf[fd]
	params := paramNames(fd)
	isParam := func(e ast.Expr) bool {
		id, ok := e.(*ast.Ident)
		if !ok {
			return false
		}
		for _, p := range params {
			if p == id.Name {
				return true
			}
		}
		return false
	}
	// every SetMethod argument of the function, for requests that set the
	// method in a separate statement
	var looseMethods []ast.Expr
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SetMethod" && len(c.Args) == 1 {
				looseMethods = append(looseMethods, c.Args[0])
			}
		}
		return true
	})

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "SetURL" || len(c.Args) != 1 {
			return true
		}
		methodExpr, urlExpr := chainSetters(c)
		if methodExpr == nil && len(looseMethods) == 1 {
			methodExpr = looseMethods[0]
		}
		if methodExpr == nil {
			out.unresolved = append(out.unresolved, fmt.Sprintf("%s: %s (no method)", service, fd.Name.Name))
			return true
		}
		envs := []map[string]string{nil}
		if isParam(methodExpr) || isParam(urlExpr) || usesParam(urlExpr, params) || usesParam(methodExpr, params) {
			envs = sc.callSiteEnvs(pkg, fd, params)
			if len(envs) == 0 {
				out.unresolved = append(out.unresolved, fmt.Sprintf("%s: %s (no call site)", service, fd.Name.Name))
				return true
			}
		}
		for _, env := range envs {
			m, ok1 := sc.resolve(methodExpr, cf, env, 0)
			p, ok2 := sc.resolve(urlExpr, cf, env, 0)
			if !ok1 || !ok2 {
				out.unresolved = append(out.unresolved, fmt.Sprintf("%s: %s (unresolved)", service, fd.Name.Name))
				continue
			}
			add(service, m, p)
		}
		return true
	})
}

func usesParam(e ast.Expr, params []string) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			for _, p := range params {
				if p == id.Name {
					found = true
				}
			}
		}
		return true
	})
	return found
}

// callSiteEnvs finds the calls of fd in its package and binds the resolvable
// arguments to the parameter names.
func (sc *coverageScanner) callSiteEnvs(pkg *coveragePkg, fd *ast.FuncDecl, params []string) []map[string]string {
	var envs []map[string]string
	for _, cf := range pkg.files {
		ast.Inspect(cf.file, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch f := c.Fun.(type) {
			case *ast.Ident:
				name = f.Name
			case *ast.SelectorExpr:
				name = f.Sel.Name
			}
			if name != fd.Name.Name || len(c.Args) != len(params) {
				return true
			}
			env := map[string]string{}
			for i, a := range c.Args {
				if v, ok := sc.resolve(a, cf, nil, 0); ok {
					env[params[i]] = v
				}
			}
			envs = append(envs, env)
			return true
		})
	}
	return envs
}

var placeholderRe = regexp.MustCompile(`\{[^}]*\}`)

// probe sends the endpoint to the fake server; handled is false for the
// router's default 404 page and for 405.
func probe(t *testing.T, fs *FakeServer, e sdkEndpoint) (handled bool, status int) {
	t.Helper()
	path := placeholderRe.ReplaceAllString(e.Path, "1")
	req, err := http.NewRequest(e.Method, fs.URL()+path, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("t12345/admin", "admin-pass")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", e.Method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return false, resp.StatusCode
	}
	if resp.StatusCode == http.StatusNotFound &&
		(strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") || resp.Header.Get(unroutedHeader) != "") {
		return false, resp.StatusCode
	}
	return true, resp.StatusCode
}

type coverageRow struct {
	sdkEndpoint
	Handled bool
}

func readGaps(t *testing.T) map[string]string {
	t.Helper()
	gaps := map[string]string{}
	b, err := os.ReadFile(coverageGapsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return gaps
		}
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, reason, _ := strings.Cut(line, "#")
		gaps[strings.TrimSpace(key)] = strings.TrimSpace(reason)
	}
	return gaps
}

// gapReasons give the reason of a known gap by path prefix; the first
// matching prefix wins.
var gapReasons = []struct{ prefix, reason string }{
	{"/service/dtm/", "digital twin manager (models, assets, definitions) is a large microservice API"},
	{"/service/dataprep/", "dataprep rule and test engine semantics are large"},
	{"/service/datahub/", "data hub (dremio jobs, offloading) is a separate microservice with its own data model"},
	{"/devicecontrol/deviceAccessToken", "needs the mTLS port and the client certificate chain, which httptest does not offer"},
	{"/tenant/statistics/files", "multipart upload of the usage statistics reports; low value"},
	{"/tenant/tenants/{tenantId}/trusted-certificates/bulk", "bulk certificate upload needs the X.509 validation of the real platform"},
	{"/tenant/trusted-certificates/verify-cert-chain", "needs the X.509 chain validation of the real platform"},
	{"/tenant/tenants/{tenantId}/applications/restricted-roles", "restricted application roles are not modeled"},
}

func gapReason(path string) string {
	for _, r := range gapReasons {
		if strings.HasPrefix(path, r.prefix) {
			return r.reason
		}
	}
	return "not implemented in the fake server"
}

func TestFakeServerCoverage(t *testing.T) {
	sc := newCoverageScanner(t)
	scan := sc.scan()
	if len(scan.endpoints) < 100 {
		t.Fatalf("only %d SDK endpoints found; the source scan is broken", len(scan.endpoints))
	}
	if len(scan.unresolved) > coverageMaxUnresolved {
		t.Fatalf("%d request builders could not be resolved (max %d), extend the scanner:\n%s",
			len(scan.unresolved), coverageMaxUnresolved, strings.Join(scan.unresolved, "\n"))
	}

	fs := New(t)
	var rows []coverageRow
	handledKeys := map[string]bool{}
	byKey := map[string]bool{}
	for _, e := range scan.endpoints {
		k := e.key()
		if _, done := byKey[k]; !done {
			ok, status := probe(t, fs, e)
			byKey[k] = ok
			if ok && os.Getenv("COVERAGE_DEBUG") != "" && (status == 404 || status >= 500 || status == 400) {
				t.Logf("DEBUG %d %s", status, k)
			}
		}
		rows = append(rows, coverageRow{e, byKey[k]})
		handledKeys[k] = byKey[k]
	}

	gaps := readGaps(t)
	var unhandled []string
	for k, ok := range handledKeys {
		if !ok {
			unhandled = append(unhandled, k)
		}
	}
	sort.Strings(unhandled)

	if os.Getenv("UPDATE") != "" {
		writeGaps(t, unhandled, gaps)
		writeCoverageDoc(t, rows, scan, gapsFor(unhandled, gaps))
		return
	}

	for _, k := range unhandled {
		if _, ok := gaps[k]; !ok {
			t.Errorf("SDK endpoint %q is not handled by the fake server and not in %s: implement it in internal/pkg/fakeserver, or (last resort) add it to the allowlist with UPDATE=1", k, coverageGapsFile)
		}
	}
	for k := range gaps {
		if ok, found := handledKeys[k]; !found {
			t.Errorf("allowlist entry %q is not an SDK endpoint (any more): remove it from %s", k, coverageGapsFile)
		} else if ok {
			t.Errorf("allowlist entry %q is handled by the fake server now: remove it from %s (UPDATE=1)", k, coverageGapsFile)
		}
	}

	// docs/FAKESERVER_COVERAGE.md must be current
	want := renderCoverageDoc(rows, scan, gapsFor(unhandled, gaps))
	got, err := os.ReadFile(coverageDocFile)
	if err != nil || string(got) != want {
		t.Errorf("%s is out of date: regenerate it with UPDATE=1 go test ./internal/pkg/fakeserver -run TestFakeServerCoverage", coverageDocFile)
	}
}

const (
	coverageMaxUnresolved  = 0
	devicesAccessTokenPath = "/devicecontrol/deviceAccessToken" // devices.ApiDeviceControlAccessToken
)

func gapsFor(unhandled []string, old map[string]string) map[string]string {
	out := map[string]string{}
	for _, k := range unhandled {
		if r, ok := old[k]; ok && r != "" {
			out[k] = r
			continue
		}
		_, path, _ := strings.Cut(k, " ")
		out[k] = gapReason(path)
	}
	return out
}

func writeGaps(t *testing.T, unhandled []string, old map[string]string) {
	t.Helper()
	reasons := gapsFor(unhandled, old)
	var b strings.Builder
	b.WriteString("# Known gaps of the fake server: SDK endpoints without a handler.\n")
	b.WriteString("# The list may only shrink. Implement the endpoint and remove its line.\n")
	b.WriteString("# Regenerate with: UPDATE=1 go test ./internal/pkg/fakeserver -run TestFakeServerCoverage\n")
	b.WriteString("# Format: METHOD /path # reason\n")
	for _, k := range unhandled {
		fmt.Fprintf(&b, "%s # %s\n", k, reasons[k])
	}
	if err := os.MkdirAll(filepath.Dir(coverageGapsFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(coverageGapsFile, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeCoverageDoc(t *testing.T, rows []coverageRow, scan coverageScan, reasons map[string]string) {
	t.Helper()
	if err := os.WriteFile(coverageDocFile, []byte(renderCoverageDoc(rows, scan, reasons)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func renderCoverageDoc(rows []coverageRow, scan coverageScan, reasons map[string]string) string {
	uniq := map[string]bool{}
	handled := map[string]bool{}
	for _, r := range rows {
		uniq[r.key()] = true
		if r.Handled {
			handled[r.key()] = true
		}
	}
	var b strings.Builder
	b.WriteString("# Fake server coverage\n\n")
	b.WriteString("<!-- Generated by internal/pkg/fakeserver/coverage_test.go. Do not edit.\n")
	b.WriteString("     Regenerate with: UPDATE=1 go test ./internal/pkg/fakeserver -run TestFakeServerCoverage -->\n\n")
	b.WriteString("Which REST endpoints the typed services of `pkg/c8y/api` call, and whether the in-memory fake server\n")
	b.WriteString("(`internal/pkg/fakeserver`, binary `cmd/c8y-fakeserver`) routes them. The endpoints are read from the\n")
	b.WriteString("`SetMethod` and `SetURL` calls of the request builders; \"handled\" means the fake server answers with\n")
	b.WriteString("something other than its default 404 page or a 405. The gaps are an allowlist\n")
	b.WriteString("(`internal/pkg/fakeserver/testdata/fakeserver_coverage_gaps.txt`) that may only shrink: the test fails\n")
	b.WriteString("for a new SDK endpoint that is neither handled nor listed.\n\n")
	fmt.Fprintf(&b, "**%d of %d distinct endpoints (method and path) are handled.**\n\n", len(handled), len(uniq))
	b.WriteString("| Service | Method | Path | Handled |\n|---|---|---|---|\n")
	for _, r := range rows {
		yes := "yes"
		if !r.Handled {
			yes = "no"
		}
		fmt.Fprintf(&b, "| %s | %s | `%s` | %s |\n", r.Service, r.Method, r.Path, yes)
	}
	b.WriteString("\n## Known gaps\n\n| Endpoint | Reason |\n|---|---|\n")
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		m, p, _ := strings.Cut(k, " ")
		fmt.Fprintf(&b, "| %s `%s` | %s |\n", m, p, reasons[k])
	}
	return b.String()
}
