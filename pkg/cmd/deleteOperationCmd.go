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

type deleteOperationCmd struct {
	*baseCmd
}

func newDeleteOperationCmd() *deleteOperationCmd {
	ccmd := &deleteOperationCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete operation/s",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteOperation,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Operation id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteOperationCmd) deleteOperation(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("devicecontrol/operations/{id}", pathParameters)

	return n.doDeleteOperation("DELETE", path, queryValue, body)
}

func (n *deleteOperationCmd) doDeleteOperation(method string, path string, query string, body map[string]interface{}) error {
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
