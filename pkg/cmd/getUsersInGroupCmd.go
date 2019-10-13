package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getUsersInGroupCmd struct {
	*baseCmd
}

func newGetUsersInGroupCmd() *getUsersInGroupCmd {
	ccmd := &getUsersInGroupCmd{}

	cmd := &cobra.Command{
		Use:   "getGroupMembership",
		Short: "Get all users in a group",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.getUsersInGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group ID")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getUsersInGroupCmd) getUsersInGroup(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/user/{tenant}/groups/{groupId}", pathParameters)

	return n.doGetUsersInGroup("GET", path, queryValue, body)
}

func (n *getUsersInGroupCmd) doGetUsersInGroup(method string, path string, query string, body map[string]interface{}) error {
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
