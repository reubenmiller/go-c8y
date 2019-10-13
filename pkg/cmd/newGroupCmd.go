package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type newGroupCmd struct {
	*baseCmd
}

func newNewGroupCmd() *newGroupCmd {
	ccmd := &newGroupCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("name", "", "Group name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newGroupCmd) newGroup(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("name"); err == nil && v != "" {
		body["name"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups", pathParameters)

	return n.doNewGroup("POST", path, queryValue, body)
}

func (n *newGroupCmd) doNewGroup(method string, path string, query string, body map[string]interface{}) error {
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
