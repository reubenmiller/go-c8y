package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getOperationCmd struct {
	*baseCmd
}

func newGetOperationCmd() *getOperationCmd {
	ccmd := &getOperationCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get operation/s",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getOperation,
	}

	cmd.Flags().String("id", "", "Operation id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getOperationCmd) getOperation(cmd *cobra.Command, args []string) error {

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

	return n.doGetOperation("GET", path, queryValue, body)
}

func (n *getOperationCmd) doGetOperation(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method: method,
			Path:   path,
			Query:  query,
			Body:   body,
		})

	if resp != nil && resp.JSONData != nil {
		fmt.Println(*resp.JSONData)
	}
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
