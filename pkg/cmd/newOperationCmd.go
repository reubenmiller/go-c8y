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

type newOperationCmd struct {
	*baseCmd
}

func newNewOperationCmd() *newOperationCmd {
	ccmd := &newOperationCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new operation",
		Long:  `Create a new operation`,
		Example: `
        
		`,
		RunE: ccmd.newOperation,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Identifies the target device on which this operation should be performed. (required)")
	cmd.Flags().String("status", "", "Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING. (required)")
	cmd.Flags().String("description", "", "Text description of the operation.")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("status")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newOperationCmd) newOperation(cmd *cobra.Command, args []string) error {

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
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("device"); err == nil && v != "" {
		body["deviceId"] = v
	}
	if v, err := cmd.Flags().GetString("status"); err == nil && v != "" {
		body["status"] = v
	}
	if v, err := cmd.Flags().GetString("description"); err == nil && v != "" {
		body["description"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("devicecontrol/operations", pathParameters)

	return n.doNewOperation("POST", path, queryValue, body)
}

func (n *newOperationCmd) doNewOperation(method string, path string, query string, body map[string]interface{}) error {
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
