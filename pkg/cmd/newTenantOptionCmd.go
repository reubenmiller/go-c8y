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

type newTenantOptionCmd struct {
	*baseCmd
}

func newNewTenantOptionCmd() *newTenantOptionCmd {
	ccmd := &newTenantOptionCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "New tenant option",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newTenantOption,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("category", "", "Category of option (required)")
	cmd.Flags().String("key", "", "Key of option (required)")
	cmd.Flags().String("value", "", "Value of option (required)")

	// Required flags
	cmd.MarkFlagRequired("category")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("value")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newTenantOptionCmd) newTenantOption(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("category"); err == nil && v != "" {
		body["category"] = v
	}
	if v, err := cmd.Flags().GetString("key"); err == nil && v != "" {
		body["key"] = v
	}
	if v, err := cmd.Flags().GetString("value"); err == nil && v != "" {
		body["value"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/tenant/options", pathParameters)

	return n.doNewTenantOption("POST", path, queryValue, body)
}

func (n *newTenantOptionCmd) doNewTenantOption(method string, path string, query string, body map[string]interface{}) error {
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
