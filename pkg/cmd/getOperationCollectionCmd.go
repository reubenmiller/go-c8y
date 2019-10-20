// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type getOperationCollectionCmd struct {
	*baseCmd
}

func newGetOperationCollectionCmd() *getOperationCollectionCmd {
	ccmd := &getOperationCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a collection of operations based on filter parameters",
		Long:  `Get a collection of operations based on filter parameters`,
		Example: `
        Get a list of pending operations
c8y operation get --status PENDING
		`,
		RunE: ccmd.getOperationCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("agent", []string{""}, "Agent ID")
	cmd.Flags().StringSlice("device", []string{""}, "Device ID")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of operation.")
	cmd.Flags().String("dateTo", "", "End date or date and time of operation.")
	cmd.Flags().String("status", "", "Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING.")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getOperationCollectionCmd) getOperationCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	agentValue := getFormattedDeviceSlice(cmd, args, "agent")
	if len(agentValue) > 0 {
		for _, item := range agentValue {
			if item != "" {
				query.Add("agentId", newIDValue(item).GetID())
			}
		}
	}
	deviceValue := getFormattedDeviceSlice(cmd, args, "device")
	if len(deviceValue) > 0 {
		for _, item := range deviceValue {
			if item != "" {
				query.Add("deviceId", newIDValue(item).GetID())
			}
		}
	}
	if v, err := cmd.Flags().GetString("dateFrom"); err == nil {
		if v != "" {
			query.Add("dateFrom", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateTo"); err == nil {
		if v != "" {
			query.Add("dateTo", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("status"); err == nil {
		if v != "" {
			query.Add("status", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
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
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("devicecontrol/operations", pathParameters)

	return n.doGetOperationCollection("GET", path, queryValue, body)
}

func (n *getOperationCollectionCmd) doGetOperationCollection(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method: method,
			Path:   path,
			Query:  query,
			Body:   body,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		if globalFlagPrettyPrint {
			fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
		} else {
			fmt.Printf("%s\n", *resp.JSONData)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
