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
	)
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

func isKnownSection(name string) bool {
	for _, section := range SectionOrder {
		if strings.EqualFold(strings.TrimSpace(name), section) {
			return true
		}
	}
	return false
}
