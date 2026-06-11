package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
)

func cmdRun(args []string) error {
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		manifestFlag = flags.String("f", "", "Path to the tenant manifest file (default: tenant.yaml)")
		only         = flags.String("only", "", "Comma-separated list of sections to apply (default: all). Sections: "+strings.Join(SectionOrder, ", "))
		dryRun       = flags.Bool("dry-run", false, "Preview the changes without applying them")
		dryRunAlias  = flags.Bool("dry", false, "Alias for --dry-run")
		force        = flags.Bool("force", false, "Replace existing version binaries and configuration files")
		concurrency  = flags.Int("concurrency", 5, "Number of concurrent software version uploads (1-20)")
		verbose      = flags.Bool("verbose", false, "Enable detailed logging")
		debug        = flags.Bool("debug", false, "Enable debug mode (verbose logging + HTTP debug)")

		allChildren     = flags.Bool("all-children", false, "Apply to all child tenants of the current tenant (overrides the manifest targets section)")
		targetSelector  = flags.String("target-selector", "", "Apply to tenants matching criteria, e.g. domain=*.example.com,company=ACME (overrides the manifest targets section)")
		includeCurrent  = flags.Bool("include-current", false, "Also include the current tenant when other target flags are used")
		credentialsMode = flags.String("credentials-mode", "", "How credentials for other tenants are obtained: "+CredentialsModeServiceUser+" or "+CredentialsModeSessions+" (overrides the manifest)")
	)
	var targetFlags stringListFlag
	flags.Var(&targetFlags, "target", "Apply to a tenant referenced by ID or domain; repeatable or comma-separated (overrides the manifest targets section)")
	flags.StringVar(manifestFlag, "manifest", "", "Alias for -f")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tenant-sync run [flags] [manifest.yaml]\n\nApply a manifest to the tenant (authentication via C8Y_* environment variables).\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *dryRunAlias {
		*dryRun = true
	}
	configureLogging(*verbose, *debug)

	manifestPath, err := manifestPathFromArgs(*manifestFlag, flags.Args())
	if err != nil {
		return err
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	targets, err := targetsFromFlags(targetFlags, *allChildren, *targetSelector, *includeCurrent)
	if err != nil {
		return err
	}
	targets = mergeTargetOverrides(manifest.Targets, targets, *credentialsMode)
	if err := targets.Validate(); err != nil {
		return fmt.Errorf("invalid targets: %w", err)
	}

	var onlySections []string
	if *only != "" {
		onlySections = strings.Split(*only, ",")
		for _, section := range onlySections {
			if !isKnownSection(section) {
				return fmt.Errorf("unknown section %q. Valid sections: %s", strings.TrimSpace(section), strings.Join(SectionOrder, ", "))
			}
		}
	}

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║        Tenant Sync for Cumulocity        ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("📄 Manifest: %s\n", manifestPath)
	if *dryRun {
		fmt.Println("🔎 Mode: dry-run (no changes will be made)")
	}
	fmt.Println()

	ctx := context.Background()

	var client *api.Client
	if !*dryRun {
		client = api.NewClientFromEnvironment(api.ClientOptions{})
		if *debug {
			client.SetDebug(true)
		}
	}

	// Work directory for downloaded artifacts
	workDir, err := os.MkdirTemp("", "tenant-sync-*")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	syncer := &Syncer{
		Client:      client,
		Resolver:    NewSourceResolver(filepath.Dir(manifestPath), workDir, *dryRun),
		DryRun:      *dryRun,
		Force:       *force,
		Concurrency: *concurrency,
		Targets:     targets,
	}

	startTime := time.Now()
	if err := syncer.Apply(ctx, manifest, onlySections); err != nil {
		return err
	}

	syncer.PrintSummary(time.Since(startTime))

	if _, failures := syncer.Summary(); len(failures) > 0 {
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("\n✓ Dry run complete - no changes made")
	}
	return nil
}

// stringListFlag collects the values of a repeatable flag
type stringListFlag []string

func (f *stringListFlag) String() string { return strings.Join(*f, ",") }

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// targetsFromFlags builds a targets spec from the CLI flags. It returns nil
// when no target flag was given (the manifest targets section then applies).
func targetsFromFlags(targets []string, allChildren bool, selector string, includeCurrent bool) (*TargetsSpec, error) {
	if len(targets) == 0 && !allChildren && selector == "" && !includeCurrent {
		return nil, nil
	}

	spec := &TargetsSpec{AllChildren: allChildren}
	for _, entry := range targets {
		for _, part := range strings.Split(entry, ",") {
			if part = strings.TrimSpace(part); part != "" {
				spec.Tenants = append(spec.Tenants, part)
			}
		}
	}
	if selector != "" {
		sel := &TenantSelector{}
		for _, pair := range strings.Split(selector, ",") {
			key, value, found := strings.Cut(pair, "=")
			if !found {
				return nil, fmt.Errorf("invalid --target-selector entry %q: expected key=value", pair)
			}
			switch strings.TrimSpace(key) {
			case "domain":
				sel.Domain = strings.TrimSpace(value)
			case "company":
				sel.Company = strings.TrimSpace(value)
			default:
				return nil, fmt.Errorf("unknown --target-selector key %q: supported keys are domain and company", strings.TrimSpace(key))
			}
		}
		spec.Selector = sel
	}
	if includeCurrent {
		include := true
		spec.Current = &include
	}
	return spec, nil
}

// mergeTargetOverrides combines the manifest targets section with the CLI
// overrides: any target selection flag replaces the manifest's selection
// entirely (but keeps its credentials config), and --credentials-mode
// overrides the credentials mode. Returns nil when nothing overrides the
// manifest.
func mergeTargetOverrides(fromManifest, fromFlags *TargetsSpec, credentialsMode string) *TargetsSpec {
	spec := fromFlags
	if spec != nil && fromManifest != nil {
		spec.Credentials = fromManifest.Credentials
	}

	if credentialsMode != "" {
		if spec == nil {
			if fromManifest != nil {
				clone := *fromManifest
				spec = &clone
			} else {
				spec = &TargetsSpec{}
			}
		}
		if spec.Credentials != nil {
			creds := *spec.Credentials
			creds.Mode = credentialsMode
			spec.Credentials = &creds
		} else {
			spec.Credentials = &TargetCredentials{Mode: credentialsMode}
		}
	}
	return spec
}

func isKnownSection(name string) bool {
	for _, section := range SectionOrder {
		if strings.EqualFold(strings.TrimSpace(name), section) {
			return true
		}
	}
	return false
}
