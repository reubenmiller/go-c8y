// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
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
	cmd.Flags().String("username", "", "Username (required)")
	cmd.Flags().String("role", "", "Role name (required)")

	// Required flags
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("role")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteRoleFromUserCmd) deleteRoleFromUser(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if cmd.Flags().Changed("pageSize") {
		if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
			query.Add("pageSize", fmt.Sprintf("%d", v))
		}
	}

	if cmd.Flags().Changed("withTotalPages") {
		if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
			query.Add("withTotalPages", "true")
		}
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)
	if v := getTenantWithDefaultFlag(cmd, "tenant", client.TenantName); v != "" {
		pathParameters["tenant"] = v
	}
	if v, err := cmd.Flags().GetString("username"); err == nil {
		if v != "" {
			pathParameters["username"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "username", err))
	}
	if v, err := cmd.Flags().GetString("role"); err == nil {
		if v != "" {
			pathParameters["role"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "role", err))
	}

	path := replacePathParameters("/user/{tenant}/users/{username}/roles/{role}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doDeleteRoleFromUser("DELETE", path, queryValue, body.GetMap(), filters)
}

func (n *deleteRoleFromUserCmd) doDeleteRoleFromUser(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
			DryRun:       globalFlagDryRun,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "")
		} else {
			responseText = []byte(*resp.JSONData)
		}

		if globalFlagPrettyPrint && json.Valid(responseText) {
			fmt.Printf("%s", pretty.Pretty(responseText))
		} else {
			fmt.Printf("%s", responseText)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
