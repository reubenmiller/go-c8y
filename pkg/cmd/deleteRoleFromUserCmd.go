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

type deleteRoleFromUserCmd struct {
	*baseCmd
}

func newDeleteRoleFromUserCmd() *deleteRoleFromUserCmd {
	ccmd := &deleteRoleFromUserCmd{}

	cmd := &cobra.Command{
		Use:   "deleteRoleFromUser",
		Short: "Unassign/Remove role from a user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteRoleFromUser,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "Username")
	cmd.Flags().String("role", "", "Role name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteRoleFromUserCmd) deleteRoleFromUser(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("username"); err == nil {
		pathParameters["username"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("role"); err == nil {
		pathParameters["role"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/users/{username}/roles/{role}", pathParameters)

	return n.doDeleteRoleFromUser("DELETE", path, queryValue, body)
}

func (n *deleteRoleFromUserCmd) doDeleteRoleFromUser(method string, path string, query string, body map[string]interface{}) error {
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
