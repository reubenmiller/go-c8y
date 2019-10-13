package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type newBinaryCmd struct {
	*baseCmd
}

func newNewBinaryCmd() *newBinaryCmd {
	ccmd := &newBinaryCmd{}

	cmd := &cobra.Command{
		Use:   "createBinary",
		Short: "New inventory binary",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newBinary,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newBinaryCmd) newBinary(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("file"); err == nil && v != "" {
		body["file"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/inventory/binaries", pathParameters)

	return n.doNewBinary("POST", path, queryValue, body)
}

func (n *newBinaryCmd) doNewBinary(method string, path string, query string, body map[string]interface{}) error {
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
