// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type getManagedObjectChildAssetCollectionCmd struct {
	*baseCmd
}

func newGetManagedObjectChildAssetCollectionCmd() *getManagedObjectChildAssetCollectionCmd {
	ccmd := &getManagedObjectChildAssetCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "listChildAssets",
		Short: "Get a collection of managedObjects child references",
		Long:  `Get a collection of managedObjects child references`,
		Example: `
$ c8y inventoryReferences listChildAssets --device 12345
Get a list of the child devices of an existing device
		`,
		RunE: ccmd.getManagedObjectChildAssetCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device.")
	cmd.Flags().StringSlice("group", []string{""}, "Group.")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getManagedObjectChildAssetCollectionCmd) getManagedObjectChildAssetCollection(cmd *cobra.Command, args []string) error {

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

	// form data
	formData := make(map[string]io.Reader)

	// body
	body := mapbuilder.NewMapBuilder()

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
				pathParameters["id"] = newIDValue(item).GetID()
			}
		}
	}
	if cmd.Flags().Changed("group") {
		groupInputValues, groupValue, err := getFormattedDeviceGroupSlice(cmd, args, "group")

		if err != nil {
			return newUserError("no matching device groups found", groupInputValues, err)
		}

		if len(groupValue) == 0 {
			return newUserError("no matching device groups found", groupInputValues)
		}

		for _, item := range groupValue {
			if item != "" {
				pathParameters["id"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("inventory/managedObjects/{id}/childAssets", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	req := c8y.RequestOptions{
		Method:       "GET",
		Path:         path,
		Query:        queryValue,
		Body:         body.GetMap(),
		FormData:     formData,
		IgnoreAccept: false,
		DryRun:       globalFlagDryRun,
	}

	return n.doGetManagedObjectChildAssetCollection(req, filters)
}

func (n *getManagedObjectChildAssetCollectionCmd) doGetManagedObjectChildAssetCollection(req c8y.RequestOptions, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		req,
	)

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "references.managedObject")
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
