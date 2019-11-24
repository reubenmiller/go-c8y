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

type newMeasurementCmd struct {
	*baseCmd
}

func newNewMeasurementCmd() *newMeasurementCmd {
	ccmd := &newMeasurementCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new measurement",
		Long:  `Create a new measurement`,
		Example: `

		`,
		RunE: ccmd.newMeasurement,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject which is the source of this measurement. (required)")
	cmd.Flags().String("time", "", "Time of the measurement. (required)")
	cmd.Flags().String("type", "", "The most specific type of this entire measurement. (required)")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("time")
	cmd.MarkFlagRequired("type")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newMeasurementCmd) newMeasurement(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
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
	body.SetMap(getDataFlag(cmd))
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
				body.Set("source.id", newIDValue(item).GetID())
			}
		}
	}
	if cmd.Flags().Changed("time") {
		if v, err := tryGetTimestampFlag(cmd, "time"); err == nil && v != "" {
			body.Set("time", decodeC8yTimestamp(v))
		} else {
			return newUserError("invalid date format", err)
		}
	}
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			body.Set("type", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "type", err))
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("measurement/measurements", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doNewMeasurement("POST", path, queryValue, body.GetMap(), filters)
}

func (n *newMeasurementCmd) doNewMeasurement(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
