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

type updateUserCurrentCmd struct {
	*baseCmd
}

func newUpdateUserCurrentCmd() *updateUserCurrentCmd {
	ccmd := &updateUserCurrentCmd{}

	cmd := &cobra.Command{
		Use:   "getCurrentUser",
		Short: "Update the current user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateUserCurrent,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("firstName", "", "User first name")
	cmd.Flags().String("lastName", "", "User last name")
	cmd.Flags().String("phone", "", "User phone number. Format: '+[country code][number]', has to be a valid MSISDN")
	cmd.Flags().String("email", "", "User email address")
	cmd.Flags().String("enabled", "", "User activation status (true/false)")
	cmd.Flags().String("password", "", "User password. Min: 6, max: 32 characters. Only Latin1 chars allowed")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateUserCurrentCmd) updateUserCurrent(cmd *cobra.Command, args []string) error {

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
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("firstName"); err == nil && v != "" {
		body["firstName"] = v
	}
	if v, err := cmd.Flags().GetString("lastName"); err == nil && v != "" {
		body["lastName"] = v
	}
	if v, err := cmd.Flags().GetString("phone"); err == nil && v != "" {
		body["phone"] = v
	}
	if v, err := cmd.Flags().GetString("email"); err == nil && v != "" {
		body["email"] = v
	}
	if v, err := cmd.Flags().GetString("enabled"); err == nil && v != "" {
		body["enabled"] = v
	}
	if v, err := cmd.Flags().GetString("password"); err == nil && v != "" {
		body["password"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/user/currentUser", pathParameters)

	return n.doUpdateUserCurrent("PUT", path, queryValue, body)
}

func (n *updateUserCurrentCmd) doUpdateUserCurrent(method string, path string, query string, body map[string]interface{}) error {
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
		if globalFlagPrettyPrint {
			fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
		} else {
			fmt.Printf("%s\n", *resp.JSONData)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
