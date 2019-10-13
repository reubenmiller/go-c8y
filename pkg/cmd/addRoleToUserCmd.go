package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type addRoleToUserCmd struct {
	*baseCmd
}

func newAddRoleToUserCmd() *addRoleToUserCmd {
	ccmd := &addRoleToUserCmd{}

	cmd := &cobra.Command{
		Use:   "addRoleTouser",
		Short: "Add role to a user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.addRoleToUser,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "Username")
	cmd.Flags().String("role", "", "")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *addRoleToUserCmd) addRoleToUser(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("role"); err == nil && v != "" {
		if _, exists := body["role"]; !exists {
			body["role"] = make(map[string]interface{})
		}
		body["role"].(map[string]interface{})["self"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("username"); err == nil {
		pathParameters["username"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/users/{username}/roles", pathParameters)

	return n.doAddRoleToUser("POST", path, queryValue, body)
}

func (n *addRoleToUserCmd) doAddRoleToUser(method string, path string, query string, body map[string]interface{}) error {
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
