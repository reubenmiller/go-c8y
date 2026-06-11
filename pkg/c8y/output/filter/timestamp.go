package filter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/karrick/tparse/v2"
)

// parseTimestamp parses the timestamp values accepted by the date filter
// operators, with the same semantics as go-c8y-cli's
// pkg/timestamp.ParseTimestamp. In order it accepts:
//
//	'<date>' [+|-] <duration>   quoted absolute date with optional offset
//	<unix_seconds> +|- <duration>
//	<duration>                  relative to now, e.g. "-25h", "1d"
//	<date>                      any dateparse-supported absolute timestamp
func parseTimestamp(value string) (time.Time, error) {
	if result, err := parseQuotedDateWithOffset(value); err == nil {
		return *result, nil
	} else if strings.HasPrefix(strings.TrimSpace(value), "'") {
		// Looks like a quoted expression but failed to parse — surface the error
		return time.Time{}, err
	}

	if result, err := parseUnixTimestampWithOffset(value); err == nil {
		return *result, nil
	}

	if value == "now" {
		return time.Now(), nil
	}
	if result, err := tparse.AddDuration(time.Now(), value); err == nil {
		return result, nil
	}

	return dateparse.ParseAny(value)
}

// normalizeOffset combines an outer sign character ('+' or '-') with a raw
// duration string (which may itself start with '+' or '-') into a single
// tparse-compatible offset, applying standard sign arithmetic:
//
//	outer='+', inner='-' → '-'   outer='-', inner='-' → '+'
//
// Spaces within the duration part are also removed.
func normalizeOffset(outerSign, durationPart string) string {
	durationPart = strings.ReplaceAll(strings.TrimSpace(durationPart), " ", "")
	if strings.HasPrefix(durationPart, "-") || strings.HasPrefix(durationPart, "+") {
		innerSign := string(durationPart[0])
		durationPart = durationPart[1:]
		if outerSign == innerSign {
			outerSign = "+"
		} else {
			outerSign = "-"
		}
	}
	return outerSign + durationPart
}

// parseQuotedDateWithOffset parses expressions of the form:
//
//	'<date>' [+|-] <duration>
//
// The date inside single quotes is parsed as an absolute timestamp and the
// optional trailing offset (e.g. "- 1h", "+2d") is applied with tparse.
func parseQuotedDateWithOffset(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "'") {
		return nil, fmt.Errorf("not a quoted date expression")
	}

	closeIdx := strings.Index(trimmed[1:], "'")
	if closeIdx == -1 {
		return nil, fmt.Errorf("unclosed single quote in date expression")
	}
	closeIdx++ // make relative to trimmed

	dateStr := trimmed[1:closeIdx]
	remainder := strings.TrimSpace(trimmed[closeIdx+1:])

	baseTime, err := dateparse.ParseAny(dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", dateStr, err)
	}

	if remainder == "" {
		return &baseTime, nil
	}

	if !strings.HasPrefix(remainder, "+") && !strings.HasPrefix(remainder, "-") {
		return nil, fmt.Errorf("expected +/- offset after quoted date, got: %s", remainder)
	}

	offset := normalizeOffset(string(remainder[0]), remainder[1:])
	result, err := tparse.AddDuration(baseTime, offset)
	if err != nil {
		return nil, fmt.Errorf("invalid offset %q: %w", offset, err)
	}
	return &result, nil
}

// parseUnixTimestampWithOffset parses expressions of the form:
//
//	<unix_seconds> +|- <duration>
//
// e.g. "1773403680 + 1h", "1773403680 -30m"
// A bare integer with no offset is intentionally not handled here so that
// values like YYYYMMDD fall through to dateparse unchanged.
func parseUnixTimestampWithOffset(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)

	end := 0
	for end < len(trimmed) && trimmed[end] >= '0' && trimmed[end] <= '9' {
		end++
	}
	if end == 0 {
		return nil, fmt.Errorf("not a unix timestamp expression")
	}

	remainder := strings.TrimSpace(trimmed[end:])

	// Require an explicit +/- offset — bare integers fall through to existing handling
	if remainder == "" || (!strings.HasPrefix(remainder, "+") && !strings.HasPrefix(remainder, "-")) {
		return nil, fmt.Errorf("not a unix timestamp with offset expression")
	}

	unixSec, err := strconv.ParseInt(trimmed[:end], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid unix timestamp %q: %w", trimmed[:end], err)
	}

	baseTime := time.Unix(unixSec, 0).UTC()

	offset := normalizeOffset(string(remainder[0]), remainder[1:])
	result, err := tparse.AddDuration(baseTime, offset)
	if err != nil {
		return nil, fmt.Errorf("invalid offset %q: %w", offset, err)
	}
	return &result, nil
}
