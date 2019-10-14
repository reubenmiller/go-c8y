// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type addRoleToGroupCmd struct {
	*baseCmd
}

func newAddRoleToGroupCmd() *addRoleToGroupCmd {
	ccmd := &addRoleToGroupCmd{}

	cmd := &cobra.Command{
		Use:   "addRoleToGroup",
		Short: "Add role to a group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.addRoleToGroup,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group ID (required)")
	cmd.Flags().String("role", "", "User role id (required)")

	// Required flags
	cmd.MarkFlagRequired("groupId")
	cmd.MarkFlagRequired("role")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *addRoleToGroupCmd) addRoleToGroup(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("groupId"); err == nil {
		pathParameters["groupId"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups/{groupId}/roles", pathParameters)

	return n.doAddRoleToGroup("POST", path, queryValue, body)
}

func (n *addRoleToGroupCmd) doAddRoleToGroup(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method: method,
			Path:   path,
			Query:  query,
			Body:   body,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
