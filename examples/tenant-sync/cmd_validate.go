package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func cmdValidate(args []string) error {
	flags := flag.NewFlagSet("validate", flag.ExitOnError)
	var (
		manifestFlag = flags.String("f", "", "Path to the tenant manifest file (default: tenant.yaml)")
		checkSources = flags.Bool("check-sources", false, "Also resolve every source (local paths must exist, GitHub releases must have matching assets)")
		verbose      = flags.Bool("verbose", false, "Enable detailed logging")
	)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tenant-sync validate [flags] [manifest.yaml]\n\nValidate a manifest file without connecting to Cumulocity.\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	configureLogging(*verbose, false)

	manifestPath, err := manifestPathFromArgs(*manifestFlag, flags.Args())
	if err != nil {
		return err
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	fmt.Printf("✓ %s is valid\n\n", manifestPath)
	printSectionCount(SectionTenantOptions, len(manifest.TenantOptions))
	printSectionCount(SectionFeatures, len(manifest.Features))
	printSectionCount(SectionApplications, len(manifest.Applications))
	printSectionCount(SectionUserGroups, len(manifest.UserGroups))
	printSectionCount(SectionUsers, len(manifest.Users))
	printSectionCount(SectionSoftware, len(manifest.Software))
	printSectionCount(SectionFirmware, len(manifest.Firmware))
	printSectionCount(SectionConfiguration, len(manifest.Configuration))
	printSectionCount(SectionSmartRest, len(manifest.SmartRestTemplates))
	printSectionCount(SectionDeviceProfiles, len(manifest.DeviceProfiles))

	if !*checkSources {
		return nil
	}

	fmt.Println("\nChecking sources...")
	resolver := NewSourceResolver(filepath.Dir(manifestPath), "", true)

	failures := 0
	check := func(section string, index int, source Source) {
		label := fmt.Sprintf("%s[%d]", section, index)
		files, err := resolver.Resolve(source)
		if err != nil {
			failures++
			fmt.Printf("  ✗ %s: %v\n", label, err)
			return
		}
		fmt.Printf("  ✓ %s: %d file(s)\n", label, len(files))
	}

	for i, spec := range manifest.Software {
		check(SectionSoftware, i, spec.Source)
	}
	for i, spec := range manifest.Firmware {
		check(SectionFirmware, i, spec.Source)
	}
	for i, spec := range manifest.Configuration {
		check(SectionConfiguration, i, spec.Source)
	}
	for i, spec := range manifest.SmartRestTemplates {
		check(SectionSmartRest, i, spec.resolvedSource())
	}

	if failures > 0 {
		return fmt.Errorf("%d source(s) failed to resolve", failures)
	}
	fmt.Println("\n✓ All sources resolved")
	return nil
}

func printSectionCount(section string, count int) {
	fmt.Printf("  %-19s %d item(s)\n", section+":", count)
}
