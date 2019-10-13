package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getBinaryCollectionCmd struct {
	*baseCmd
}

func newGetBinaryCollectionCmd() *getBinaryCollectionCmd {
	ccmd := &getBinaryCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getBinaryCollection",
		Short: "Get collection of inventory binaries",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.getBinaryCollection,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getBinaryCollectionCmd) getBinaryCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/inventory/binaries", pathParameters)

	return n.doGetBinaryCollection("GET", path, queryValue, body)
}

func (n *getBinaryCollectionCmd) doGetBinaryCollection(method string, path string, query string, body map[string]interface{}) error {
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
