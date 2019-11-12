// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type newManagedObjectChildAssetCmd struct {
	*baseCmd
}

func newNewManagedObjectChildAssetCmd() *newManagedObjectChildAssetCmd {
	ccmd := &newManagedObjectChildAssetCmd{}

	cmd := &cobra.Command{
		Use:   "createChildAsset",
		Short: "Create a child asset (device or devicegroup) reference",
		Long:  `Create a child asset (device or devicegroup) reference`,
		Example: `
        
		`,
		RunE: ccmd.newManagedObjectChildAsset,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("group", []string{""}, "Group id (required)")
	cmd.Flags().StringSlice("newChildDevice", []string{""}, "new child device asset")
	cmd.Flags().StringSlice("newChildGroup", []string{""}, "new child device group asset")

	// Required flags
	cmd.MarkFlagRequired("group")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newManagedObjectChildAssetCmd) newManagedObjectChildAsset(cmd *cobra.Command, args []string) error {

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
	if cmd.Flags().Changed("newChildDevice") {
		newChildDeviceInputValues, newChildDeviceValue, err := getFormattedDeviceSlice(cmd, args, "newChildDevice")

		if err != nil {
			return newUserError("no matching devices found", newChildDeviceInputValues, err)
		}

		if len(newChildDeviceValue) == 0 {
			return newUserError("no matching devices found", newChildDeviceInputValues)
		}

		for _, item := range newChildDeviceValue {
			if item != "" {
				body.Set("managedObject.id", newIDValue(item).GetID())
			}
		}
	}
	if cmd.Flags().Changed("newChildGroup") {
		newChildGroupInputValues, newChildGroupValue, err := getFormattedDeviceGroupSlice(cmd, args, "newChildGroup")

		if err != nil {
			return newUserError("no matching device groups found", newChildGroupInputValues, err)
		}

		if len(newChildGroupValue) == 0 {
			return newUserError("no matching device groups found", newChildGroupInputValues)
		}

		for _, item := range newChildGroupValue {
			if item != "" {
				body.Set("managedObject.id", newIDValue(item).GetID())
			}
		}
	}

	// path parameters
	pathParameters := make(map[string]string)
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

	return n.doNewManagedObjectChildAsset("POST", path, queryValue, body.GetMap(), filters)
}

func (n *newManagedObjectChildAssetCmd) doNewManagedObjectChildAsset(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
		log.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

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
