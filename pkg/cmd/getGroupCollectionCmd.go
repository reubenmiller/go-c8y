package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getGroupCollectionCmd struct {
	*baseCmd
}

func newGetGroupCollectionCmd() *getGroupCollectionCmd {
	ccmd := &getGroupCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getCollection",
		Short: "Get collection of (user) groups",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.getGroupCollection,
	}

	cmd.Flags().String("tenant", "", "Tenant")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGroupCollectionCmd) getGroupCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups", pathParameters)

	return n.doGetGroupCollection("GET", path, queryValue, body)
}

func (n *getGroupCollectionCmd) doGetGroupCollection(method string, path string, query string, body map[string]interface{}) error {
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
