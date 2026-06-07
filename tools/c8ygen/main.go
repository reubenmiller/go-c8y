// Command c8ygen generates the Layer-0 substrate of the go-c8y v2 SDK (path constants,
// enums) from the Cumulocity OpenAPI specification, and reports drift between the spec
// and the hand-written SDK.
//
// Usage:
//
//	c8ygen generate [--spec PATH | --fetch] [--out DIR]
//	c8ygen lint     [--spec PATH | --fetch] [--src DIR] [--strict]
//	c8ygen fetch    [--url URL] [--out PATH]
//
// The spec is read from docs/c8y-oas.yml by default. Pass --fetch to download the
// latest spec from cumulocity.com instead. See docs/API_GEN.md for the design.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "generate", "gen":
		err = cmdGenerate(args)
	case "lint":
		err = cmdLint(args)
	case "fetch":
		err = cmdFetch(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `c8ygen — generate the v2 SDK Layer-0 substrate from the Cumulocity OpenAPI spec

Commands:
  generate   Emit path/enum constants into the spec package
  lint       Report drift between the OAS and the SDK source
  fetch      Download the latest OpenAPI spec

Run "c8ygen <command> -h" for command flags.
`)
}

// loadFlags adds the shared spec-source flags to a flag set.
func loadFlags(fs *flag.FlagSet) *LoadOptions {
	opt := &LoadOptions{}
	fs.StringVar(&opt.Path, "spec", DefaultSpecPath, "path to the local OpenAPI spec")
	fs.BoolVar(&opt.Fetch, "fetch", false, "download the latest spec instead of reading --spec")
	fs.StringVar(&opt.URL, "url", DefaultSpecURL, "spec URL (used with --fetch)")
	return opt
}

func cmdGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	opt := loadFlags(fs)
	out := fs.String("out", "pkg/c8y/api/spec", "output directory for generated files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	doc, _, err := Load(*opt)
	if err != nil {
		return err
	}
	source := specSource(*opt)
	res, err := Generate(doc, source, *out)
	if err != nil {
		return err
	}
	fmt.Printf("generated %d path constants and %d enums from %s\n", res.Paths, res.Enums, source)
	for _, f := range res.Files {
		fmt.Printf("  wrote %s\n", f)
	}
	return nil
}

func cmdLint(args []string) error {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	opt := loadFlags(fs)
	src := fs.String("src", "pkg/c8y/api", "SDK source root to scan for path literals")
	strict := fs.Bool("strict", false, "exit non-zero when drift is detected")
	if err := fs.Parse(args); err != nil {
		return err
	}

	doc, _, err := Load(*opt)
	if err != nil {
		return err
	}
	res, err := Lint(doc, *src)
	if err != nil {
		return err
	}
	PrintLintReport(res)
	if *strict && res.HasDrift() {
		return fmt.Errorf("drift detected (%d missing, %d extra)", len(res.MissingInSDK), len(res.ExtraInSDK))
	}
	return nil
}

func cmdFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	url := fs.String("url", DefaultSpecURL, "spec URL to download")
	out := fs.String("out", DefaultSpecPath, "destination file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, raw, err := Load(LoadOptions{Fetch: true, URL: *url})
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", *out, err)
	}
	fmt.Printf("downloaded %d bytes to %s\n", len(raw), *out)
	return nil
}

// specSource returns a stable, human-readable description of where the spec came from,
// for the generated-file header. The label is normalized so the output is identical
// whether the generator is invoked from the repo root (task generate) or from the spec
// package directory (go generate) — leading "../" / "./" hops are stripped.
func specSource(opt LoadOptions) string {
	if opt.Fetch {
		if opt.URL != "" {
			return opt.URL
		}
		return DefaultSpecURL
	}
	path := opt.Path
	if path == "" {
		path = DefaultSpecPath
	}
	return stripParentHops(path)
}

// stripParentHops removes leading "../" and "./" components so a relative spec path
// resolves to a stable repo-relative label.
func stripParentHops(path string) string {
	for {
		switch {
		case strings.HasPrefix(path, "../"):
			path = path[3:]
		case strings.HasPrefix(path, "./"):
			path = path[2:]
		default:
			return path
		}
	}
}
