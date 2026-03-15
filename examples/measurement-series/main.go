package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
)

func main() {
	outputFormat := flag.String("output", "csv", "Output format: csv or json")
	debug := flag.Bool("debug", false, "Enable debug logging")

	source := flag.String("source", "ext:c8y_Serial:rmi_demo_param", "Source device: direct ID, name:<name>, ext:<type>:<id>, or query:<query>")
	dateFrom := flag.String("date-from", "", "Start of time range (RFC3339, e.g. 2026-03-01T00:00:00Z). Defaults to 20 days ago")
	dateTo := flag.String("date-to", "", "End of time range (RFC3339). Defaults to now")
	interval := flag.String("interval", "1h", "Aggregation interval (e.g. 1h, 30m, 1d)")
	aggFunctions := flag.String("agg", "avg,min,max", "Comma-separated aggregation functions (avg,min,max,count,sum,stdDevPop,stdDevSamp)")
	series := flag.String("series", "cgroup-mosquitto.memory,cgroup-mosquitto.percent-cpu", "Comma-separated series to filter (e.g. c8y_Temperature.T)")
	revert := flag.Bool("revert", false, "Return results in reverse chronological order")

	flag.Parse()

	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	if *debug {
		client.SetDebugWithAuth(true)
	}

	from := time.Now().Add(-20 * 24 * time.Hour)
	if *dateFrom != "" {
		var err error
		from, err = time.Parse(time.RFC3339, *dateFrom)
		if err != nil {
			slog.Error("Invalid --date-from value", "err", err)
			os.Exit(1)
		}
	}

	to := time.Now()
	if *dateTo != "" {
		var err error
		to, err = time.Parse(time.RFC3339, *dateTo)
		if err != nil {
			slog.Error("Invalid --date-to value", "err", err)
			os.Exit(1)
		}
	}

	opts := measurements.ListSeriesOptions{
		DateFrom:            from,
		DateTo:              to,
		AggregationFunction: strings.Split(*aggFunctions, ","),
		AggregationInterval: *interval,
		Revert:              *revert,
		Source:              managedobjects.DeviceRef(*source),
	}
	if *series != "" {
		opts.Series = strings.Split(*series, ",")
	}

	seriesResult := client.Measurements.ListSeries(context.Background(), opts)

	// Always check for errors
	if seriesResult.Err != nil {
		slog.Error("Could not get measurement series", "err", seriesResult.Err)
		os.Exit(1)
	}

	slog.Info("Response", "status", seriesResult.HTTPStatus, "duration", seriesResult.Duration)

	switch *outputFormat {
	case "json":
		// JSON Lines output - one JSON object per timestamp
		enc := json.NewEncoder(os.Stdout)
		for _, obj := range seriesResult.Data.ToJSONRows() {
			enc.Encode(obj)
		}
	default:
		// CSV output - each stat column is "<type>.<name>.<stat>"
		columns, flatRows := seriesResult.Data.ToFlatRows()
		fmt.Println("time,source.id,source.name," + strings.Join(columns, ","))
		for _, row := range flatRows {
			vals := make([]string, len(row.Values))
			for i, v := range row.Values {
				if v != nil {
					vals[i] = strconv.FormatFloat(*v, 'f', -1, 64)
				}
			}
			fmt.Printf("%s,%s,%s,%s\n", row.Time.Format(time.RFC3339), row.DeviceID, row.DeviceName, strings.Join(vals, ","))
		}
	}
}
