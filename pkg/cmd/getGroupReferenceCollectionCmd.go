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

type getGroupReferenceCollectionCmd struct {
	*baseCmd
}

func newGetGroupReferenceCollectionCmd() *getGroupReferenceCollectionCmd {
	ccmd := &getGroupReferenceCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "listReferences",
		Short: "Get information about all groups of a user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getGroupReferenceCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "Username (required)")

	// Required flags
	cmd.MarkFlagRequired("username")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGroupReferenceCollectionCmd) getGroupReferenceCollection(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/user/{tenant}/users/{username}/groups", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doGetGroupReferenceCollection("GET", path, queryValue, body.GetMap(), filters)
}

func (n *getGroupReferenceCollectionCmd) doGetGroupReferenceCollection(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
			responseText = filters.Apply(*resp.JSONData, "todo")
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
