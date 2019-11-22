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

type deleteManagedObjectChildAssetReferenceCmd struct {
	*baseCmd
}

func newDeleteManagedObjectChildAssetReferenceCmd() *deleteManagedObjectChildAssetReferenceCmd {
	ccmd := &deleteManagedObjectChildAssetReferenceCmd{}

	cmd := &cobra.Command{
		Use:   "deleteChildAsset",
		Short: "Delete child asset reference",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteManagedObjectChildAssetReference,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("asset", []string{""}, "Asset id (required)")
	cmd.Flags().StringSlice("childDevice", []string{""}, "Child device")
	cmd.Flags().StringSlice("childGroup", []string{""}, "Child device group")

	// Required flags
	cmd.MarkFlagRequired("asset")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteManagedObjectChildAssetReferenceCmd) deleteManagedObjectChildAssetReference(cmd *cobra.Command, args []string) error {

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

	// path parameters
	pathParameters := make(map[string]string)
	if cmd.Flags().Changed("asset") {
		assetInputValues, assetValue, err := getFormattedDeviceSlice(cmd, args, "asset")

		if err != nil {
			return newUserError("no matching devices found", assetInputValues, err)
		}

		if len(assetValue) == 0 {
			return newUserError("no matching devices found", assetInputValues)
		}

		for _, item := range assetValue {
			if item != "" {
				pathParameters["asset"] = newIDValue(item).GetID()
			}
		}
	}
	if cmd.Flags().Changed("childDevice") {
		childDeviceInputValues, childDeviceValue, err := getFormattedDeviceSlice(cmd, args, "childDevice")

		if err != nil {
			return newUserError("no matching devices found", childDeviceInputValues, err)
		}

		if len(childDeviceValue) == 0 {
			return newUserError("no matching devices found", childDeviceInputValues)
		}

		for _, item := range childDeviceValue {
			if item != "" {
				pathParameters["childDevice"] = newIDValue(item).GetID()
			}
		}
	}
	if cmd.Flags().Changed("childGroup") {
		childGroupInputValues, childGroupValue, err := getFormattedDeviceGroupSlice(cmd, args, "childGroup")

		if err != nil {
			return newUserError("no matching device groups found", childGroupInputValues, err)
		}

		if len(childGroupValue) == 0 {
			return newUserError("no matching device groups found", childGroupInputValues)
		}

		for _, item := range childGroupValue {
			if item != "" {
				pathParameters["childGroup"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("inventory/managedObjects/{asset}/childAssets/{reference}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doDeleteManagedObjectChildAssetReference("DELETE", path, queryValue, body.GetMap(), filters)
}

func (n *deleteManagedObjectChildAssetReferenceCmd) doDeleteManagedObjectChildAssetReference(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
