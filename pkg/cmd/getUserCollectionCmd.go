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

type getUserCollectionCmd struct {
	*baseCmd
}

func newGetUserCollectionCmd() *getUserCollectionCmd {
	ccmd := &getUserCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a collection of users based on filter parameters",
		Long:  `Get a collection of users based on filter parameters`,
		Example: `
        retrieve users, where username starts with 'js', and every user belongs to one of the groups 2, 3 or 4, and the owner is 'admin', and is not a device user.
c8y user list --username js --groups 2,3,4 --owner admin
		`,
		RunE: ccmd.getUserCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "prefix or full username")
	cmd.Flags().String("groups", "", "numeric group identifiers separated by commas; result will contain only users which belong to at least one of specified groups")
	cmd.Flags().String("owner", "", "exact username")
	cmd.Flags().Bool("onlyDevices", false, "If set to 'true', result will contain only users created during bootstrap process (starting with 'device_'). If flag is absent (or false) the result will not contain 'device_' users.")
	cmd.Flags().Bool("withSubusersCount", false, "if set to 'true', then each of returned users will contain additional field 'subusersCount' - number of direct subusers (users with corresponding 'owner').")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getUserCollectionCmd) getUserCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("username"); err == nil {
		if v != "" {
			query.Add("username", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("groups"); err == nil {
		if v != "" {
			query.Add("groups", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("owner"); err == nil {
		if v != "" {
			query.Add("owner", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("onlyDevices"); err == nil {
		if v {
			query.Add("onlyDevices", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("withSubusersCount"); err == nil {
		if v {
			query.Add("withSubusersCount", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
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

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "tenant", err))
	}

	path := replacePathParameters("/user/{tenant}/users", pathParameters)

	return n.doGetUserCollection("GET", path, queryValue, body)
}

func (n *getUserCollectionCmd) doGetUserCollection(method string, path string, query string, body map[string]interface{}) error {
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
