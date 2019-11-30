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

type newExternalIDCmd struct {
	*baseCmd
}

func newNewExternalIDCmd() *newExternalIDCmd {
	ccmd := &newExternalIDCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new external id",
		Long:  `Create a new external id`,
		Example: `
$ c8y identity create --device 1234 --type test --name myserialnumber
Create external identity
		`,
		RunE: ccmd.newExternalID,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject linked to the external ID. (required)")
	cmd.Flags().String("type", "", "The type of the external identifier as string, e.g. 'com_cumulocity_model_idtype_SerialNumber'. (required)")
	cmd.Flags().String("name", "", "The identifier used in the external system that Cumulocity interfaces with. (required)")

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newExternalIDCmd) newExternalID(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			body.Set("type", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "type", err))
	}
	if v, err := cmd.Flags().GetString("name"); err == nil {
		if v != "" {
			body.Set("externalId", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "name", err))
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

	path := replacePathParameters("identity/globalIds/{device}/externalIds", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doNewExternalID("POST", path, queryValue, body.GetMap(), filters)
}

func (n *newExternalIDCmd) doNewExternalID(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
