package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// patternsFlag is a custom flag type that allows multiple pattern values
type patternsFlag []string

func (p *patternsFlag) String() string {
	if len(*p) == 0 {
		return "*"
	}
	return strings.Join(*p, ", ")
}

func (p *patternsFlag) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func main() {
	// Parse command-line flags
	var (
		dir          = flag.String("dir", "", "Directory to search for software packages (required)")
		patterns     patternsFlag
		softwareType = flag.String("type", "", "Software type (e.g., 'firmware', 'application')")
		concurrency  = flag.Int("concurrency", 5, "Number of concurrent uploads")
		dryRun       = flag.Bool("dry-run", false, "Preview what would be uploaded without actually uploading")
		verbose      = flag.Bool("verbose", false, "Enable detailed logging")
		debug        = flag.Bool("debug", false, "Enable debug mode (verbose logging + HTTP debug)")
		force        = flag.Bool("force", false, "Force replacement of existing versions (deletes old binary and uploads new one)")
		noProgress   = flag.Bool("no-progress", false, "Disable progress bar (automatic in non-TTY environments)")
	)

	flag.Var(&patterns, "pattern", "Glob pattern for matching files (can be specified multiple times, default: *)")

	flag.Parse()

	// Configure logging (debug implies verbose)
	if *debug || *verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		})))
	}

	// Set default pattern if none specified
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}

	// Validate required flags
	if *dir == "" {
		fmt.Fprintf(os.Stderr, "Error: --dir flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate directory exists
	dirInfo, err := os.Stat(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: directory not found: %s\n", *dir)
		os.Exit(1)
	}
	if !dirInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", *dir)
		os.Exit(1)
	}

	// Initialize Cumulocity client
	ctx := context.Background()

	var client *c8y_api.Client
	if !*dryRun {
		client = c8y_api.NewClientFromEnvironment(c8y_api.ClientOptions{})

		// Enable HTTP debug logging on the underlying resty client if debug mode is enabled
		if *debug {
			client.Client.SetDebug(true)
		}
	}

	// Print header
	printHeader()

	// Scan directory for matching files
	fmt.Printf("🔍 Scanning directory: %s\n", *dir)
	files, err := findMatchingFiles(*dir, patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("❌ No files found matching pattern(s): %s\n", patterns.String())
		os.Exit(0)
	}

	fmt.Printf("📦 Found %d file(s) matching pattern(s): %s\n\n", len(files), patterns.String())

	// Parse software information from filenames
	var softwareInfos []*SoftwareInfo
	for _, file := range files {
		info, err := ParseSoftwareFromFilename(file, *softwareType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", file, err)
			continue
		}

		if err := ValidateSoftwareInfo(info); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", file, err)
			continue
		}

		softwareInfos = append(softwareInfos, info)
	}

	if len(softwareInfos) == 0 {
		fmt.Printf("❌ No valid software packages found\n")
		os.Exit(0)
	}

	// Create summary
	summary := CreateSummary(softwareInfos)

	// Print upload plan
	printUploadPlan(summary)

	if *dryRun {
		fmt.Println("\n✓ Dry run complete - no changes made")
		os.Exit(0)
	}

	// Create upload config
	config := &UploadConfig{
		Client:       client,
		Concurrency:  *concurrency,
		SoftwareType: *softwareType,
		DryRun:       *dryRun,
		Force:        *force,
	}

	// Upload software versions with progress  tracking
	startTime := time.Now()

	// Determine if progress bar should be shown
	// Disable if: explicitly disabled via flag OR output is not a TTY (e.g., CI environment)
	showProgress := !*noProgress && term.IsTerminal(int(os.Stdout.Fd()))

	fmt.Println("📤 Uploading versions...")
	bar := progressbar.NewOptions(len(softwareInfos),
		progressbar.OptionSetDescription("Uploading"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionSetVisibility(showProgress),
	)

	result := UploadSoftwareVersions(ctx, config, softwareInfos, func(completed, total int, currentFile string) {
		bar.Set(completed)
	})

	bar.Finish()
	fmt.Println()

	// Print results
	elapsed := time.Since(startTime)
	printResults(result, elapsed)

	// Exit with appropriate code
	if result.FailureCount > 0 {
		os.Exit(1)
	}
}

// findMatchingFiles finds all files matching any of the patterns in the directory (recursively)
func findMatchingFiles(dir string, patterns []string) ([]string, error) {
	var matches []string
	seen := make(map[string]bool) // Track unique files

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if file matches any pattern
		filename := filepath.Base(path)
		for _, pattern := range patterns {
			matched, err := filepath.Match(pattern, filename)
			if err != nil {
				return err
			}

			if matched && !seen[path] {
				matches = append(matches, path)
				seen[path] = true
				break // Don't check remaining patterns for this file
			}
		}

		return nil
	})

	return matches, err
}

// printHeader prints the application header
func printHeader() {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   Software Package Uploader for C8Y      ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
}

// printUploadPlan prints a summary of what will be uploaded
func printUploadPlan(summary *SoftwareSummary) {
	fmt.Println("📋 Upload Plan:")

	// Sort software names for consistent output
	names := make([]string, 0, len(summary.Groups))
	for name := range summary.Groups {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, key := range names {
		versions := summary.Groups[key]
		if len(versions) == 0 {
			continue
		}

		versionStrs := make([]string, len(versions))
		for i, v := range versions {
			versionStrs[i] = v.Version
		}

		// Sort versions
		sort.Strings(versionStrs)

		// Get actual software name and architecture from first version
		actualName := versions[0].Name
		arch := versions[0].Architecture
		archSuffix := ""
		if arch != "" {
			archSuffix = fmt.Sprintf(" [%s]", arch)
		}

		// Print software and its versions
		if len(versionStrs) <= 3 {
			fmt.Printf("  • %s%s: %d version(s) [%s]\n", actualName, archSuffix, len(versions), strings.Join(versionStrs, ", "))
		} else {
			fmt.Printf("  • %s%s: %d version(s) [%s, ... and %d more]\n",
				actualName, archSuffix, len(versions), strings.Join(versionStrs[:3], ", "), len(versionStrs)-3)
		}
	}

	fmt.Printf("\n📊 Summary: %d software package(s), %d version(s) total\n\n",
		summary.TotalSoftware, summary.TotalVersions)
}

// printResults prints the upload results
func printResults(result *UploadResult, elapsed time.Duration) {
	if result.SuccessCount > 0 {
		fmt.Printf("✅ Successfully processed %d version(s)\n", result.SuccessCount)
		if result.VersionsCreated > 0 {
			fmt.Printf("   📤 Newly uploaded: %d\n", result.VersionsCreated)
		}
		if result.VersionsReplaced > 0 {
			fmt.Printf("   🔄 Replaced: %d\n", result.VersionsReplaced)
		}
		if result.VersionsFound > 0 {
			fmt.Printf("   ♻️  Already existed: %d\n", result.VersionsFound)
		}
	}

	if result.FailureCount > 0 {
		fmt.Printf("❌ Failed to upload %d version(s):\n", result.FailureCount)
		for _, err := range result.Errors {
			if err.Name != "" && err.Version != "" {
				fmt.Printf("  • %s v%s: %v\n", err.Name, err.Version, err.Error)
			} else if err.FilePath != "" {
				fmt.Printf("  • %s: %v\n", err.FilePath, err.Error)
			} else {
				fmt.Printf("  • %v\n", err.Error)
			}
		}
	}

	fmt.Printf("\n⏱️  Total time: %.1fs\n", elapsed.Seconds())

	// Print summary
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Printf("Total: %d processed (%d new, %d replaced, %d existing), %d failed\n",
		result.SuccessCount, result.VersionsCreated, result.VersionsReplaced, result.VersionsFound, result.FailureCount)
}
