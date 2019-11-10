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

type newEventCmd struct {
	*baseCmd
}

func newNewEventCmd() *newEventCmd {
	ccmd := &newEventCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create event",
		Long:  `Create event`,
		Example: `
        
		`,
		RunE: ccmd.newEvent,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject which is the source of this event. (required)")
	cmd.Flags().String("time", "", "Time of the event. (required)")
	cmd.Flags().String("type", "", "Identifies the type of this event. (required)")
	cmd.Flags().String("text", "", "Text description of the event. (required)")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("time")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("text")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newEventCmd) newEvent(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("time"); err == nil {
		if v != "" {
			body.Set("time", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "time", err))
	}
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			body.Set("type", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "type", err))
	}
	if v, err := cmd.Flags().GetString("text"); err == nil {
		if v != "" {
			body.Set("text", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "text", err))
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("event/events", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doNewEvent("POST", path, queryValue, body.GetMap(), filters)
}

func (n *newEventCmd) doNewEvent(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
