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

type deleteMeasurementCollectionCmd struct {
	*baseCmd
}

func newDeleteMeasurementCollectionCmd() *deleteMeasurementCollectionCmd {
	ccmd := &deleteMeasurementCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "deleteCollection",
		Short: "Delete a collection of measurements",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteMeasurementCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device ID")
	cmd.Flags().String("type", "", "Measurement type.")
	cmd.Flags().String("valueFragmentType", "", "value fragment type")
	cmd.Flags().String("valueFragmentSeries", "", "value fragment series")
	cmd.Flags().String("fragmentType", "", "Fragment name from measurement.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of measurement occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of measurement occurrence.")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteMeasurementCollectionCmd) deleteMeasurementCollection(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			query.Add("type", url.QueryEscape(v))
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "type", err))
	}
	if v, err := cmd.Flags().GetString("valueFragmentType"); err == nil {
		if v != "" {
			query.Add("valueFragmentType", url.QueryEscape(v))
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "valueFragmentType", err))
	}
	if v, err := cmd.Flags().GetString("valueFragmentSeries"); err == nil {
		if v != "" {
			query.Add("valueFragmentSeries", url.QueryEscape(v))
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "valueFragmentSeries", err))
	}
	if v, err := cmd.Flags().GetString("fragmentType"); err == nil {
		if v != "" {
			query.Add("fragmentType", url.QueryEscape(v))
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "fragmentType", err))
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

	path := replacePathParameters("measurement/measurements", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doDeleteMeasurementCollection("DELETE", path, queryValue, body.GetMap(), filters)
}

func (n *deleteMeasurementCollectionCmd) doDeleteMeasurementCollection(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
