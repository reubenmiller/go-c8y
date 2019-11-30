// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
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
$ c8y measurements getSeries -source 12345 --series nx_WEA_29_Delta.MDL10FG001 --series nx_WEA_29_Delta.ST9 --dateFrom "-10min"
Get a list of series [nx_WEA_29_Delta.MDL10FG001] and [nx_WEA_29_Delta.ST9] for device 12345
		`,
		RunE: ccmd.getMeasurementSeries,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device ID")
	cmd.Flags().StringSlice("series", []string{""}, "measurement type and series name, e.g. c8y_AccelerationMeasurement.acceleration")
	cmd.Flags().String("aggregationType", "", "Fragment name from measurement.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of measurement occurrence. (required)")
	cmd.Flags().String("dateTo", "0s", "End date or date and time of measurement occurrence.")

	// Required flags
	cmd.MarkFlagRequired("dateFrom")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getMeasurementSeriesCmd) getMeasurementSeries(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if cmd.Flags().Changed("device") {
		deviceInputValues, deviceValue, err := getFormattedDeviceSlice(cmd, args, "device")

		if err != nil {
			return newUserError("no matching devices found", deviceInputValues, err)
		}

		if len(deviceValue) == 0 {
			return newUserError("no matching devices found", deviceInputValues)
		}

		for _, item := range deviceValue {
			if item != "" {
				query.Add("source", newIDValue(item).GetID())
			}
		}
	}
	if items, err := cmd.Flags().GetStringSlice("series"); err == nil {
		if len(items) > 0 {
			for _, v := range items {
				if v != "" {
					query.Add("series", url.QueryEscape(v))
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
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "aggregationType", err))
	}
	if cmd.Flags().Changed("dateFrom") {
		if v, err := tryGetTimestampFlag(cmd, "dateFrom"); err == nil && v != "" {
			query.Add("dateFrom", v)
		} else {
			return newUserError("invalid date format", err)
		}
	}
	if cmd.Flags().Changed("dateTo") {
		if v, err := tryGetTimestampFlag(cmd, "dateTo"); err == nil && v != "" {
			query.Add("dateTo", v)
		} else {
			return newUserError("invalid date format", err)
		}
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
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("measurement/measurements/series", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doGetMeasurementSeries("GET", path, queryValue, body.GetMap(), filters)
}

func (n *getMeasurementSeriesCmd) doGetMeasurementSeries(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
			DryRun:       globalFlagDryRun,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "")
		} else {
			responseText = []byte(*resp.JSONData)
		}

		if globalFlagPrettyPrint && json.Valid(responseText) {
			fmt.Printf("%s", pretty.Pretty(responseText))
		} else {
			fmt.Printf("%s", responseText)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
