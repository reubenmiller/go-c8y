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

type updateOperationCmd struct {
	*baseCmd
}

func newUpdateOperationCmd() *updateOperationCmd {
	ccmd := &updateOperationCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update operation",
		Long:  `Update operation`,
		Example: `
        
		`,
		RunE: ccmd.updateOperation,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Operation id")
	cmd.Flags().String("status", "", "Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING. (required)")
	cmd.Flags().String("failureReason", "", "Reason for the failure. Use whne setting status to FAILED")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("status")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateOperationCmd) updateOperation(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("status"); err == nil && v != "" {
		body["status"] = v
	}
	if v, err := cmd.Flags().GetString("failureReason"); err == nil && v != "" {
		body["failureReason"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "id", err))
	}

	path := replacePathParameters("devicecontrol/operations/{id}", pathParameters)

	return n.doUpdateOperation("PUT", path, queryValue, body)
}

func (n *updateOperationCmd) doUpdateOperation(method string, path string, query string, body map[string]interface{}) error {
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
