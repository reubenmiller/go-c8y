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

type newTenantCmd struct {
	*baseCmd
}

func newNewTenantCmd() *newTenantCmd {
	ccmd := &newTenantCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "New tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("company", "", "Company name. Maximum 256 characters (required)")
	cmd.Flags().String("domain", "", "Domain name to be used for the tenant. Maximum 256 characters (required)")
	cmd.Flags().String("id", "", "The tenant ID. Will be auto-generated if not present.")
	cmd.Flags().String("adminName", "", "Username of the tenant administrator")
	cmd.Flags().String("adminPass", "", "Password of the tenant administrator")
	cmd.Flags().String("contactName", "", "A contact name, for example an administrator, of the tenant")
	cmd.Flags().String("contact_phone", "", "An international contact phone number")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("company")
	cmd.MarkFlagRequired("domain")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newTenantCmd) newTenant(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("company"); err == nil && v != "" {
		body["company"] = v
	}
	if v, err := cmd.Flags().GetString("domain"); err == nil && v != "" {
		body["domain"] = v
	}
	if v, err := cmd.Flags().GetString("id"); err == nil && v != "" {
		body["id"] = v
	}
	if v, err := cmd.Flags().GetString("adminName"); err == nil && v != "" {
		body["adminName"] = v
	}
	if v, err := cmd.Flags().GetString("adminPass"); err == nil && v != "" {
		body["adminPass"] = v
	}
	if v, err := cmd.Flags().GetString("contactName"); err == nil && v != "" {
		body["contactName"] = v
	}
	if v, err := cmd.Flags().GetString("contact_phone"); err == nil && v != "" {
		body["contact_phone"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/tenant/tenants", pathParameters)

	return n.doNewTenant("POST", path, queryValue, body)
}

func (n *newTenantCmd) doNewTenant(method string, path string, query string, body map[string]interface{}) error {
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
