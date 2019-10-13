package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteGroupCmd struct {
	*baseCmd
}

func newDeleteGroupCmd() *deleteGroupCmd {
	ccmd := &deleteGroupCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a new group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("id", "", "Group id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteGroupCmd) deleteGroup(cmd *cobra.Command, args []string) error {

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

	return n.doDeleteGroup("DELETE", path, queryValue, body)
}

func (n *deleteGroupCmd) doDeleteGroup(method string, path string, query string, body map[string]interface{}) error {
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
