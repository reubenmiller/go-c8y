// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	cmd.Flags().StringSlice("user", []string{""}, "User (required)")
	cmd.Flags().StringSlice("role", []string{""}, "Role name (required)")

	// Required flags
	cmd.MarkFlagRequired("user")
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

	// form data
	formData := make(map[string]io.Reader)

	// body
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)
	if v := getTenantWithDefaultFlag(cmd, "tenant", client.TenantName); v != "" {
		pathParameters["tenant"] = v
	}
	if cmd.Flags().Changed("user") {
		userInputValues, userValue, err := getFormattedUserSlice(cmd, args, "user")

		if err != nil {
			return newUserError("no matching users found", userInputValues, err)
		}

		if len(userValue) == 0 {
			return newUserError("no matching users found", userInputValues)
		}

		for _, item := range userValue {
			if item != "" {
				pathParameters["user"] = newIDValue(item).GetID()
			}
		}
	}
	if cmd.Flags().Changed("role") {
		roleInputValues, roleValue, err := getFormattedRoleSlice(cmd, args, "role")

		if err != nil {
			return newUserError("no matching roles found", roleInputValues, err)
		}

		if len(roleValue) == 0 {
			return newUserError("no matching roles found", roleInputValues)
		}

		for _, item := range roleValue {
			if item != "" {
				pathParameters["role"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("/user/{tenant}/users/{user}/roles/{role}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	req := c8y.RequestOptions{
		Method:       "DELETE",
		Path:         path,
		Query:        queryValue,
		Body:         body.GetMap(),
		FormData:     formData,
		IgnoreAccept: false,
		DryRun:       globalFlagDryRun,
	}

	return n.doDeleteRoleFromUser(req, filters)
}

func (n *deleteRoleFromUserCmd) doDeleteRoleFromUser(req c8y.RequestOptions, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		req,
	)

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

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
