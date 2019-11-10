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

type getManagedObjectChildAssetReferenceCmd struct {
	*baseCmd
}

func newGetManagedObjectChildAssetReferenceCmd() *getManagedObjectChildAssetReferenceCmd {
	ccmd := &getManagedObjectChildAssetReferenceCmd{}

	cmd := &cobra.Command{
		Use:   "getChildAsset",
		Short: "Get managed object child asset reference",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getManagedObjectChildAssetReference,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("asset", []string{""}, "Asset id (required)")
	cmd.Flags().StringSlice("reference", []string{""}, "Asset reference id (required)")

	// Required flags
	cmd.MarkFlagRequired("asset")
	cmd.MarkFlagRequired("reference")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getManagedObjectChildAssetReferenceCmd) getManagedObjectChildAssetReference(cmd *cobra.Command, args []string) error {

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
	if cmd.Flags().Changed("reference") {
		referenceInputValues, referenceValue, err := getFormattedDeviceSlice(cmd, args, "reference")

		if err != nil {
			return newUserError("no matching devices found", referenceInputValues, err)
		}

		if len(referenceValue) == 0 {
			return newUserError("no matching devices found", referenceInputValues)
		}

		for _, item := range referenceValue {
			if item != "" {
				pathParameters["reference"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("inventory/managedObjects/{asset}/childAssets/{reference}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doGetManagedObjectChildAssetReference("GET", path, queryValue, body.GetMap(), filters)
}

func (n *getManagedObjectChildAssetReferenceCmd) doGetManagedObjectChildAssetReference(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
