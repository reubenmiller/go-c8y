package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteUserFromGroupCmd struct {
	*baseCmd
}

func newDeleteUserFromGroupCmd() *deleteUserFromGroupCmd {
	ccmd := &deleteUserFromGroupCmd{}

	cmd := &cobra.Command{
		Use:   "deleteUserFromGroup",
		Short: "Delete a user from a group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteUserFromGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group ID")
	cmd.Flags().String("userId", "", "User id/username")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteUserFromGroupCmd) deleteUserFromGroup(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("groupId"); err == nil {
		pathParameters["groupId"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("userId"); err == nil {
		pathParameters["userId"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups/{groupId}/users/{userId}", pathParameters)

	return n.doDeleteUserFromGroup("DELETE", path, queryValue, body)
}

func (n *deleteUserFromGroupCmd) doDeleteUserFromGroup(method string, path string, query string, body map[string]interface{}) error {
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
