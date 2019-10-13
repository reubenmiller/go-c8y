package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getSystemOptionCollectionCmd struct {
	*baseCmd
}

func newGetSystemOptionCollectionCmd() *getSystemOptionCollectionCmd {
	ccmd := &getSystemOptionCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get collection of system options",
		Long:  `This endpoint provides a set of read-only properties pre-defined in platform configuration. The response format is exactly the same as for OptionCollection.`,
		Example: `
        
		`,
		RunE: ccmd.getSystemOptionCollection,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getSystemOptionCollectionCmd) getSystemOptionCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/tenant/system/options", pathParameters)

	return n.doGetSystemOptionCollection("GET", path, queryValue, body)
}

func (n *getSystemOptionCollectionCmd) doGetSystemOptionCollection(method string, path string, query string, body map[string]interface{}) error {
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
