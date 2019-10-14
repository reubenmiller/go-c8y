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

type deleteRoleFromGroupCmd struct {
	*baseCmd
}

func newDeleteRoleFromGroupCmd() *deleteRoleFromGroupCmd {
	ccmd := &deleteRoleFromGroupCmd{}

	cmd := &cobra.Command{
		Use:   "deleteRoleFromGroup",
		Short: "Unassign/Remove role from a group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteRoleFromGroup,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group id")
	cmd.Flags().String("role", "", "Role name, e.g. ROLE_TENANT_MANAGEMENT_ADMIN")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteRoleFromGroupCmd) deleteRoleFromGroup(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("role"); err == nil {
		pathParameters["role"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups/{groupId}/roles/{role}", pathParameters)

	return n.doDeleteRoleFromGroup("DELETE", path, queryValue, body)
}

func (n *deleteRoleFromGroupCmd) doDeleteRoleFromGroup(method string, path string, query string, body map[string]interface{}) error {
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
