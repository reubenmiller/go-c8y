package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getGroupCmd struct {
	*baseCmd
}

func newGetGroupCmd() *getGroupCmd {
	ccmd := &getGroupCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Create a new group by id",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("id", "", "Group id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGroupCmd) getGroup(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups/{id}", pathParameters)

	return n.doGetGroup("GET", path, queryValue, body)
}

func (n *getGroupCmd) doGetGroup(method string, path string, query string, body map[string]interface{}) error {
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
