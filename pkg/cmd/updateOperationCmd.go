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
	cmd.Flags().String("status", "", "Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING.")
	cmd.Flags().String("failureReason", "", "Reason for the failure. Use whne setting status to FAILED")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateOperationCmd) updateOperation(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

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
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("devicecontrol/operations/{id}", pathParameters)

	return n.doUpdateOperation("PUT", path, queryValue, body)
}

func (n *updateOperationCmd) doUpdateOperation(method string, path string, query string, body map[string]interface{}) error {
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
		fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
