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

type newManagedObjectChildDeviceCmd struct {
	*baseCmd
}

func newNewManagedObjectChildDeviceCmd() *newManagedObjectChildDeviceCmd {
	ccmd := &newManagedObjectChildDeviceCmd{}

	cmd := &cobra.Command{
		Use:   "createChildDevice",
		Short: "Create a child device reference",
		Long:  `Create a child device reference`,
		Example: `
$ c8y inventoryReferences createChildDevice --device 12345 --newChild 44235
Assign a device as a child device to an existing device
		`,
		RunE: ccmd.newManagedObjectChildDevice,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "ManagedObject id (required)")
	cmd.Flags().StringSlice("newChild", []string{""}, "new child device (required)")

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("newChild")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newManagedObjectChildDeviceCmd) newManagedObjectChildDevice(cmd *cobra.Command, args []string) error {

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
	if cmd.Flags().Changed("newChild") {
		newChildInputValues, newChildValue, err := getFormattedDeviceSlice(cmd, args, "newChild")

		if err != nil {
			return newUserError("no matching devices found", newChildInputValues, err)
		}

		if len(newChildValue) == 0 {
			return newUserError("no matching devices found", newChildInputValues)
		}

		for _, item := range newChildValue {
			if item != "" {
				body.Set("managedObject.id", newIDValue(item).GetID())
			}
		}
	}

	// path parameters
	pathParameters := make(map[string]string)
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
				pathParameters["device"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("inventory/managedObjects/{device}/childDevices", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doNewManagedObjectChildDevice("POST", path, queryValue, body.GetMap(), filters)
}

func (n *newManagedObjectChildDeviceCmd) doNewManagedObjectChildDevice(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
			responseText = filters.Apply(*resp.JSONData, "managedObject")
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
