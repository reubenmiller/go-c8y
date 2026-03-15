package cmd

// alarms.go - alarm sub-commands demonstrating how the go-c8y SDK drives a CLI.
//
// SDK TO CLI BINDING PATTERN
//
// Each command struct holds the corresponding SDK options struct (or a mirror
// with string adapters for time.Time fields). Cobra flags are registered by
// mapping each SDK field to a flag with:
//
//   - The same name as the url struct tag:  url:"severity" -> --severity
//   - The field's Go doc comment, copied verbatim as the flag Usage string
//   - The matching Go type:  string, []string, bool; time.Time via adapter
//
// This makes the SDK options structs the authoritative source of documentation:
// add a field to alarms.ListOptions, add one flag binding here, done.
//
// RELATIVE DATE STRINGS
//
// time.Time SDK fields accept both absolute RFC3339 and relative durations:
//   -10m, -1h, -7d, -2w, -1M, +30m, now
// A thin parseRelativeTime adapter is defined at the bottom of this file.
//
// PAGINATION
//
// ListAll iterates pages transparently.
//   --page-size  controls per-request page size.
//   --max-items  soft-caps the total items across all pages.
// Summary statistics come back in result.Meta and are printed to stderr so
// stdout stays clean for JSON piping.

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// alarms (parent group)
// ---------------------------------------------------------------------------

func newAlarmsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alarms",
		Short: "Interact with Cumulocity alarms",
		Long: `Manage alarms in Cumulocity IoT.

Each sub-command maps one-to-one to an alarms.Service function in the SDK.
Flag names and usage strings are derived directly from the SDK options struct
fields and their comments.`,
	}
	cmd.AddCommand(
		newAlarmsListCmd(),
		newAlarmsGetCmd(),
		newAlarmsCreateCmd(),
		newAlarmsCountCmd(),
	)
	return cmd
}

// ---------------------------------------------------------------------------
// alarms list
// ---------------------------------------------------------------------------

// listAlarmsFlags holds the CLI flag values for the list command.
// It mirrors alarms.ListOptions field-by-field, replacing time.Time fields
// with strings so users can pass relative durations like "-1h" or "-7d".
//
//	SDK field                           CLI flag     Adapter
//	──────────────────────────────────  ──────────── ──────────────────────
//	ListOptions.Source       string   → --device     none
//	ListOptions.Type         []string → --type       none
//	ListOptions.Status       []Status → --status     string -> AlarmStatus
//	ListOptions.Severity     []Sev    → --severity   string -> AlarmSeverity
//	ListOptions.Resolved     bool     → --resolved   none
//	ListOptions.DateFrom     time.T   → --dateFrom   parseRelativeTime
//	ListOptions.DateTo       time.T   → --dateTo     parseRelativeTime
//	PaginationOptions.PageSize int    → --page-size  none
//	PaginationOptions.MaxItems int64  → --max-items  none
type listAlarmsFlags struct {
	device    string
	alarmType []string
	status    []string
	severity  []string
	resolved  bool
	dateFrom  string
	dateTo    string
	pageSize  int
	maxItems  int64
}

func newAlarmsListCmd() *cobra.Command {
	f := &listAlarmsFlags{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get alarm collection",
		Long: `Get a collection of alarms based on filter parameters.

Output is streamed as JSON to stdout (one alarm per line) so it can be piped
to other tools. Summary statistics are printed to stderr.

Examples:
  c8y-demo alarms list --severity MAJOR
  c8y-demo alarms list --status ACTIVE --dateFrom "-10m"
  c8y-demo alarms list --device "name:myDevice" --max-items 100
  c8y-demo alarms list --device "ext:c8y_Serial:ABC123" --status ACTIVE`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlarmsList(f)
		},
	}

	// ── Bind SDK fields to Cobra flags ────────────────────────────────────────
	// The usage strings below are taken verbatim from the SDK field comments.
	// SDK docs == CLI docs; there is no separate documentation to maintain.

	cmd.Flags().StringVar(&f.device, "device", "",
		// alarms.ListOptions.Source:
		"Source device to filter alarms by. Accepts resolver strings:\n"+
			"  direct ID, \"name:deviceName\", \"ext:type:id\", \"query:...\"")

	cmd.Flags().StringArrayVar(&f.alarmType, "type", nil,
		// alarms.ListOptions.Type:
		"The types of alarm to search for (repeatable: --type A --type B)")

	cmd.Flags().StringArrayVar(&f.status, "status", nil,
		// alarms.ListOptions.Status:
		"Alarm status filter: ACTIVE, ACKNOWLEDGED, CLEARED (repeatable)")

	cmd.Flags().StringArrayVar(&f.severity, "severity", nil,
		// alarms.ListOptions.Severity:
		"Alarm severity filter: CRITICAL, MAJOR, MINOR, WARNING (repeatable)")

	cmd.Flags().BoolVar(&f.resolved, "resolved", false,
		// alarms.ListOptions.Resolved:
		"When true, only CLEARED alarms are returned. Takes precedence over --status")

	cmd.Flags().StringVar(&f.dateFrom, "dateFrom", "",
		// alarms.ListOptions.DateFrom (+ adapter note):
		"Start date/time of alarm occurrence.\n"+
			"Accepts RFC3339 or relative duration: -10m, -1h, -7d, -2w")

	cmd.Flags().StringVar(&f.dateTo, "dateTo", "",
		// alarms.ListOptions.DateTo (+ adapter note):
		"End date/time of alarm occurrence.\n"+
			"Accepts RFC3339 or relative duration: -10m, -1h, -7d, -2w")

	// ── Pagination flags (from embedded PaginationOptions) ───────────────────
	cmd.Flags().IntVar(&f.pageSize, "page-size", 100,
		// pagination.PaginationOptions.PageSize:
		"Maximum number of alarms per page request (Cumulocity max: 2000)")

	cmd.Flags().Int64Var(&f.maxItems, "max-items", 0,
		// pagination.PaginationOptions.MaxItems:
		"Maximum total alarms to return across all pages (0 = unlimited)")

	return cmd
}

func runAlarmsList(f *listAlarmsFlags) error {
	ctx := requestContext()
	client := clientFactory()

	// Build typed SDK options from the CLI flag values.
	// This is the bridge: raw CLI strings become SDK-typed values here.
	opts := alarms.ListOptions{
		Source:   managedobjects.DeviceRef(f.device),
		Type:     f.alarmType,
		Resolved: f.resolved,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: f.pageSize,
			MaxItems: f.maxItems,
		},
	}

	for _, s := range f.status {
		opts.Status = append(opts.Status, model.AlarmStatus(s))
	}
	for _, s := range f.severity {
		opts.Severity = append(opts.Severity, model.AlarmSeverity(s))
	}

	if f.dateFrom != "" {
		t, err := parseRelativeTime(f.dateFrom)
		if err != nil {
			return fmt.Errorf("--dateFrom: %w", err)
		}
		opts.DateFrom = t
	}
	if f.dateTo != "" {
		t, err := parseRelativeTime(f.dateTo)
		if err != nil {
			return fmt.Errorf("--dateTo: %w", err)
		}
		opts.DateTo = t
	}

	// ListAll returns an iterator that fetches the next page only when needed,
	// respecting MaxItems as a soft cap across all pages transparently.
	iter := client.Alarms.ListAll(ctx, opts)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	var count int
	for item, err := range iter.Items() {
		if err != nil {
			return fmt.Errorf("reading alarms: %w", err)
		}
		if encErr := enc.Encode(item); encErr != nil {
			return encErr
		}
		count++
	}

	if iterErr := iter.Err(); iterErr != nil {
		return fmt.Errorf("alarm iterator: %w", iterErr)
	}

	fmt.Fprintf(os.Stderr, "Returned %d alarm(s)\n", count)
	return nil
}

// ---------------------------------------------------------------------------
// alarms get
// ---------------------------------------------------------------------------

func newAlarmsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get alarm by ID",
		Long: `Retrieve a single alarm by its Cumulocity internal ID.

The alarm is printed as indented JSON to stdout.

Examples:
  c8y-demo alarms get 12345`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlarmsGet(args[0])
		},
	}
}

func runAlarmsGet(id string) error {
	ctx := requestContext()
	client := clientFactory()

	// Get returns op.Result[jsonmodels.Alarm]: a typed result carrying the data,
	// HTTP status code, error, duration, and a Meta map with server-side metadata.
	result := client.Alarms.Get(ctx, id)
	if result.Err != nil {
		return fmt.Errorf("get alarm %s: %w", id, result.Err)
	}

	// Print request metadata to stderr, keeping stdout clean for JSON piping.
	printResultMeta(result)

	// Alarm embeds jsondoc.Facade which implements json.Marshaler by returning
	// the raw response bytes directly - zero re-serialisation overhead.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result.Data)
}

// ---------------------------------------------------------------------------
// alarms create
// ---------------------------------------------------------------------------

// createAlarmsFlags mirrors alarms.CreateOptions field-by-field.
// String adapters replace time.Time for the 'time' field.
type createAlarmsFlags struct {
	device    string // CreateOptions.Source
	alarmType string // CreateOptions.Type
	text      string // CreateOptions.Text
	severity  string // CreateOptions.Severity
	status    string // CreateOptions.Status
	alarmTime string // string adapter for CreateOptions.Time
}

func newAlarmsCreateCmd() *cobra.Command {
	f := &createAlarmsFlags{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an alarm",
		Long: `Create a new alarm in Cumulocity IoT.

The created alarm is printed as JSON to stdout.

Examples:
  c8y-demo alarms create --device 12345 --type c8y_TestAlarm --text "Disk full" --severity MAJOR
  c8y-demo alarms create --device "name:myDevice" --type c8y_Temp --text "Over temp" --severity CRITICAL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlarmsCreate(f)
		},
	}

	// Bind SDK CreateOptions fields to flags.
	// Descriptions come verbatim from the alarms.CreateOptions field comments.

	cmd.Flags().StringVar(&f.device, "device", "",
		// CreateOptions.Source:
		"Source device. Accepts resolver strings:\n"+
			"  direct ID, \"name:deviceName\", \"ext:type:id\", \"query:...\" (required)")
	_ = cmd.MarkFlagRequired("device")

	cmd.Flags().StringVar(&f.alarmType, "type", "",
		// CreateOptions.Type:
		"Type of the alarm, e.g. c8y_TestAlarm (required)")
	_ = cmd.MarkFlagRequired("type")

	cmd.Flags().StringVar(&f.text, "text", "",
		// CreateOptions.Text:
		"Text description of the alarm (required)")
	_ = cmd.MarkFlagRequired("text")

	cmd.Flags().StringVar(&f.severity, "severity", "MAJOR",
		// CreateOptions.Severity:
		"Severity: CRITICAL | MAJOR | MINOR | WARNING")

	cmd.Flags().StringVar(&f.status, "status", "ACTIVE",
		// CreateOptions.Status:
		"Initial status: ACTIVE | ACKNOWLEDGED | CLEARED")

	cmd.Flags().StringVar(&f.alarmTime, "time", "",
		// CreateOptions.Time (+ adapter note):
		"Alarm occurrence time. Accepts RFC3339 or relative duration (default: now)")

	return cmd
}

func runAlarmsCreate(f *createAlarmsFlags) error {
	ctx := requestContext()
	client := clientFactory()

	opts := alarms.CreateOptions{
		Source:   managedobjects.DeviceRef(f.device),
		Type:     f.alarmType,
		Text:     f.text,
		Severity: f.severity,
		Status:   f.status,
	}

	if f.alarmTime != "" {
		t, err := parseRelativeTime(f.alarmTime)
		if err != nil {
			return fmt.Errorf("--time: %w", err)
		}
		opts.Time = t
	}

	result := client.Alarms.Create(ctx, opts)
	if result.Err != nil {
		return fmt.Errorf("create alarm: %w", result.Err)
	}

	fmt.Fprintf(os.Stderr, "Created (HTTP %d, status: %s, duration: %v)\n",
		result.HTTPStatus, result.Status, result.Duration)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result.Data)
}

// ---------------------------------------------------------------------------
// alarms count
// ---------------------------------------------------------------------------

// countAlarmsFlags uses the same filters as listAlarmsFlags but the SDK
// returns op.Result[int64] rather than a collection.
type countAlarmsFlags struct {
	device   string
	status   []string
	severity []string
	resolved bool
	dateFrom string
	dateTo   string
}

func newAlarmsCountCmd() *cobra.Command {
	f := &countAlarmsFlags{}

	cmd := &cobra.Command{
		Use:   "count",
		Short: "Count alarms matching a filter",
		Long: `Return the total number of alarms that match the given filter.

The integer count is printed to stdout. Accepts the same filter flags as
'alarms list'.

Examples:
  c8y-demo alarms count --status ACTIVE
  c8y-demo alarms count --severity CRITICAL --dateFrom "-1h"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAlarmsCount(f)
		},
	}

	cmd.Flags().StringVar(&f.device, "device", "",
		"Source device (same resolver strings as list --device)")
	cmd.Flags().StringArrayVar(&f.status, "status", nil,
		"Alarm status filter: ACTIVE, ACKNOWLEDGED, CLEARED (repeatable)")
	cmd.Flags().StringArrayVar(&f.severity, "severity", nil,
		"Alarm severity filter: CRITICAL, MAJOR, MINOR, WARNING (repeatable)")
	cmd.Flags().BoolVar(&f.resolved, "resolved", false,
		"When true, only CLEARED alarms are counted")
	cmd.Flags().StringVar(&f.dateFrom, "dateFrom", "",
		"Start date/time. Accepts RFC3339 or relative duration: -10m, -1h, -7d")
	cmd.Flags().StringVar(&f.dateTo, "dateTo", "",
		"End date/time. Accepts RFC3339 or relative duration")

	return cmd
}

func runAlarmsCount(f *countAlarmsFlags) error {
	ctx := requestContext()
	client := clientFactory()

	opts := alarms.CountOptions{
		Source:   managedobjects.DeviceRef(f.device),
		Resolved: f.resolved,
	}
	for _, s := range f.status {
		opts.Status = append(opts.Status, string(s))
	}
	for _, s := range f.severity {
		opts.Severity = append(opts.Severity, string(s))
	}
	if f.dateFrom != "" {
		t, err := parseRelativeTime(f.dateFrom)
		if err != nil {
			return fmt.Errorf("--dateFrom: %w", err)
		}
		opts.DateFrom = t
	}
	if f.dateTo != "" {
		t, err := parseRelativeTime(f.dateTo)
		if err != nil {
			return fmt.Errorf("--dateTo: %w", err)
		}
		opts.DateTo = t
	}

	// Count returns op.Result[int64]: the same rich metadata envelope as
	// collection results (HTTPStatus, Duration, Err, Meta), but Data is int64.
	result := client.Alarms.Count(ctx, opts)
	if result.Err != nil {
		return fmt.Errorf("count alarms: %w", result.Err)
	}

	fmt.Println(result.Data)
	return nil
}

// ---------------------------------------------------------------------------
// parseRelativeTime - relative duration string to time.Time
// ---------------------------------------------------------------------------

// parseRelativeTime converts a human-friendly duration string to time.Time.
//
// Supported formats (same convention as go-c8y-cli):
//
//	now                        -> time.Now()
//	-10m / +10m                -> 10 minutes ago / from now
//	-1h  / +1h                 -> 1 hour ago / from now
//	-7d  / +7d                 -> 7 days ago / from now
//	-2w  / +2w                 -> 14 days ago / from now
//	-1M                        -> 1 calendar month ago (time.AddDate)
//	2024-01-15T10:00:00Z       -> absolute RFC3339
//	2024-01-15                 -> absolute date (midnight UTC)
//
// This helper lives in this file so command files are self-contained.
func parseRelativeTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "now") {
		return time.Now(), nil
	}

	// Absolute time: no leading sign.
	if !strings.HasPrefix(s, "+") && !strings.HasPrefix(s, "-") {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unrecognised time format %q (use RFC3339 or relative: -10m, -1h, -7d)", s)
	}

	sign := 1
	if strings.HasPrefix(s, "-") {
		sign = -1
	}
	mag := s[1:] // strip leading sign

	switch {
	case strings.HasSuffix(mag, "w"):
		n, err := strconv.Atoi(strings.TrimSuffix(mag, "w"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid week duration %q", s)
		}
		return time.Now().Add(time.Duration(sign*n*7) * 24 * time.Hour), nil

	case strings.HasSuffix(mag, "M"):
		n, err := strconv.Atoi(strings.TrimSuffix(mag, "M"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid month duration %q", s)
		}
		return time.Now().AddDate(0, sign*n, 0), nil

	case strings.HasSuffix(mag, "d"):
		n, err := strconv.Atoi(strings.TrimSuffix(mag, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day duration %q", s)
		}
		return time.Now().Add(time.Duration(sign*n) * 24 * time.Hour), nil

	default:
		// Delegate s, m, h, ms, µs, ns to stdlib.
		d, err := time.ParseDuration(mag)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		return time.Now().Add(time.Duration(sign) * d), nil
	}
}

// ---------------------------------------------------------------------------
// printResultMeta prints timing/pagination metadata from op.Result to stderr.
// Keeping metadata on stderr lets stdout remain clean for JSON piping.
// ---------------------------------------------------------------------------

func printResultMeta[T any](r op.Result[T]) {
	if r.HTTPStatus != 0 {
		fmt.Fprintf(os.Stderr, "HTTP %d  duration: %v\n", r.HTTPStatus, r.Duration)
	}
	if total := r.TotalElements(); total > 0 {
		fmt.Fprintf(os.Stderr, "Total elements: %d\n", total)
	}
}
