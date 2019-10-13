package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getGroupByNameCmd struct {
	*baseCmd
}

func newGetGroupByNameCmd() *getGroupByNameCmd {
	ccmd := &getGroupByNameCmd{}

	cmd := &cobra.Command{
		Use:   "getByName",
		Short: "Get a group by name",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getGroupByName,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("name", "", "Group name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGroupByNameCmd) getGroupByName(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("name"); err == nil {
		pathParameters["name"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groupByName/{name}", pathParameters)

	return n.doGetGroupByName("GET", path, queryValue, body)
}

func (n *getGroupByNameCmd) doGetGroupByName(method string, path string, query string, body map[string]interface{}) error {
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
