package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
)

// The full annotated example manifest doubles as the init template
//
//go:embed tenant.example.yaml
var manifestTemplate string

const minimalTemplate = `# yaml-language-server: $schema=` + SchemaID + `
#
# tenant-sync manifest
#
# Validate with: tenant-sync validate %[1]s
# Preview with:  tenant-sync run -f %[1]s --dry-run
# Apply with:    tenant-sync run -f %[1]s
#
# Run 'tenant-sync init --full <path>' for a fully annotated example, or see:
# https://github.com/reubenmiller/go-c8y/tree/main/examples/tenant-sync

software:
  - source:
      path: ./packages
      patterns: ["*.deb", "*.rpm", "*.apk", "*.ipk"]

# firmware:
#   - deviceType: mydevicetype
#     source:
#       github:
#         repo: owner/repo
#         release: latest
#         assets: ["*.wic.xz"]

# configuration:
#   - name: myconfig
#     configurationType: myconfig.toml
#     source:
#       path: ./config/myconfig.toml

# tenantOptions:
#   - category: configuration
#     key: my.setting
#     value: "true"

# features:
#   - key: feature-branding

# applications:
#   - name: advanced-software-mgmt

# deviceProfiles:
#   - name: base-profile
#     deviceType: mydevicetype
#     software:
#       - name: mypackage
#         version: 1.0.0
`

func cmdInit(args []string) error {
	flags := flag.NewFlagSet("init", flag.ExitOnError)
	var (
		full  = flags.Bool("full", false, "Write the fully annotated example manifest instead of a minimal skeleton")
		force = flags.Bool("force", false, "Overwrite the file if it already exists")
	)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tenant-sync init [flags] [manifest.yaml]\n\nCreate a new tenant manifest file (default: tenant.yaml).\n\nFlags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}

	path, err := manifestPathFromArgs("", flags.Args())
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil && !*force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}

	content := fmt.Sprintf(minimalTemplate, path)
	if *full {
		content = manifestTemplate
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Sanity check: the generated manifest must validate
	if _, err := LoadManifest(path); err != nil {
		return fmt.Errorf("generated manifest failed validation (this is a bug): %w", err)
	}

	fmt.Printf("✓ Created %s\n", path)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Edit %s to describe your tenant\n", path)
	fmt.Printf("  2. Validate it:          tenant-sync validate %s\n", path)
	fmt.Printf("  3. Preview the changes:  tenant-sync run -f %s --dry-run\n", path)
	fmt.Printf("  4. Apply it:             tenant-sync run -f %s\n", path)
	return nil
}
