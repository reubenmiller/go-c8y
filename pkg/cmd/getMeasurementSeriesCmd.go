// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type getMeasurementSeriesCmd struct {
	*baseCmd
}

func newGetMeasurementSeriesCmd() *getMeasurementSeriesCmd {
	ccmd := &getMeasurementSeriesCmd{}

	cmd := &cobra.Command{
		Use:   "getSeries",
		Short: "Get a collection of measurements based on filter parameters",
		Long:  `Get a collection of measurements based on filter parameters`,
		Example: `
        Get a list of measurement series
c8y measurement getSeries

Get a list of series [nx_WEA_29_Delta.MDL10FG001] and [nx_WEA_29_Delta.ST9] for device 12345
measurement getSeries --source 12345 --series nx_WEA_29_Delta.MDL10FG001 --series nx_WEA_29_Delta.ST9 --dateFrom (Get-C8yDate (last 10min)) --dateTo (Get-C8yDate (last 0min))
		`,
		RunE: ccmd.getMeasurementSeries,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device ID")
	cmd.Flags().StringSlice("series", []string{""}, "measurement type and series name, e.g. c8y_AccelerationMeasurement.acceleration")
	cmd.Flags().String("aggregationType", "", "Fragment name from measurement.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of measurement occurrence. (required)")
	cmd.Flags().String("dateTo", "", "End date or date and time of measurement occurrence. (required)")

	// Required flags
	cmd.MarkFlagRequired("dateFrom")
	cmd.MarkFlagRequired("dateTo")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getMeasurementSeriesCmd) getMeasurementSeries(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	deviceValue := getFormattedDeviceSlice(cmd, args, "device")
	if len(deviceValue) > 0 {
		for _, item := range deviceValue {
			if item != "" {
				query.Add("source", newIDValue(item).GetID())
			}
		}
	}
	if v, err := cmd.Flags().GetStringSlice("series"); err == nil {
		if len(v) > 0 {
			for _, item := range v {
				if item != "" {
					query.Add("series", item)
				}
			}
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("aggregationType"); err == nil {
		if v != "" {
			query.Add("aggregationType", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateFrom"); err == nil {
		if v != "" {
			query.Add("dateFrom", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateTo"); err == nil {
		if v != "" {
			query.Add("dateTo", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if cmd.Flags().Changed("pageSize") {
		if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
			query.Add("pageSize", fmt.Sprintf("%d", v))
		}
	}

	if cmd.Flags().Changed("withTotalPages") {
		if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
			query.Add("withTotalPages", "true")
		}
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("measurement/measurements/series", pathParameters)

	return n.doGetMeasurementSeries("GET", path, queryValue, body)
}

func (n *getMeasurementSeriesCmd) doGetMeasurementSeries(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		if globalFlagPrettyPrint {
			fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
		} else {
			fmt.Printf("%s\n", *resp.JSONData)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
