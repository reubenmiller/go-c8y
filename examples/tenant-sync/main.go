package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const usageText = `tenant-sync - GitOps-style tenant synchronisation for Cumulocity IoT

Usage:
  tenant-sync <command> [flags]

Commands:
  init       Create a new tenant manifest file
  validate   Validate a manifest file (schema and, optionally, sources)
  run        Apply a manifest to the tenant (alias: apply)
  schema     Generate the JSON schema for the manifest file
  help       Show this help

Run 'tenant-sync <command> --help' for command-specific flags.

Examples:
  tenant-sync init
  tenant-sync validate tenant.yaml
  tenant-sync run -f tenant.yaml --dry-run
  tenant-sync run -f tenant.yaml --only firmware,deviceProfiles
  tenant-sync schema -o tenant.schema.json
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	var err error
	switch command := os.Args[1]; command {
	case "run", "apply":
		err = cmdRun(os.Args[2:])
	case "init":
		err = cmdInit(os.Args[2:])
	case "validate":
		err = cmdValidate(os.Args[2:])
	case "schema":
		err = cmdSchema(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n%s", command, usageText)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// configureLogging sets the log level based on the verbose/debug flags
func configureLogging(verbose, debug bool) {
	level := slog.LevelError
	if verbose || debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

// manifestPathFromArgs returns the manifest path from the -f flag or a single
// positional argument, defaulting to tenant.yaml when neither is given
func manifestPathFromArgs(flagValue string, args []string) (string, error) {
	if flagValue != "" && len(args) > 0 {
		return "", fmt.Errorf("specify the manifest either with -f or as an argument, not both")
	}
	if flagValue != "" {
		return flagValue, nil
	}
	switch len(args) {
	case 0:
		return "tenant.yaml", nil
	case 1:
		return args[0], nil
	default:
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(args[1:], " "))
	}
}
