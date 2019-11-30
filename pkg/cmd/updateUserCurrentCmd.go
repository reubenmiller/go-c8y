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

type updateUserCurrentCmd struct {
	*baseCmd
}

func newUpdateUserCurrentCmd() *updateUserCurrentCmd {
	ccmd := &updateUserCurrentCmd{}

	cmd := &cobra.Command{
		Use:   "updateCurrentUser",
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
	body := mapbuilder.NewMapBuilder()
	body.SetMap(getDataFlag(cmd))
	if v, err := cmd.Flags().GetString("firstName"); err == nil {
		if v != "" {
			body.Set("firstName", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "firstName", err))
	}
	if v, err := cmd.Flags().GetString("lastName"); err == nil {
		if v != "" {
			body.Set("lastName", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "lastName", err))
	}
	if v, err := cmd.Flags().GetString("phone"); err == nil {
		if v != "" {
			body.Set("phone", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "phone", err))
	}
	if v, err := cmd.Flags().GetString("email"); err == nil {
		if v != "" {
			body.Set("email", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "email", err))
	}
	if v, err := cmd.Flags().GetString("enabled"); err == nil {
		if v != "" {
			body.Set("enabled", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "enabled", err))
	}
	if v, err := cmd.Flags().GetString("password"); err == nil {
		if v != "" {
			body.Set("password", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "password", err))
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/user/currentUser", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doUpdateUserCurrent("PUT", path, queryValue, body.GetMap(), filters)
}

func (n *updateUserCurrentCmd) doUpdateUserCurrent(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
