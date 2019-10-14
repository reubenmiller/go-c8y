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

type updateTenantCmd struct {
	*baseCmd
}

func newUpdateTenantCmd() *updateTenantCmd {
	ccmd := &updateTenantCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Tenant id (required)")
	cmd.Flags().String("company", "", "Company name. Maximum 256 characters (required)")
	cmd.Flags().String("domain", "", "Domain name to be used for the tenant. Maximum 256 characters (required)")
	cmd.Flags().String("adminName", "", "Username of the tenant administrator")
	cmd.Flags().String("adminPass", "", "Password of the tenant administrator")
	cmd.Flags().String("contactName", "", "A contact name, for example an administrator, of the tenant")
	cmd.Flags().String("contact_phone", "", "An international contact phone number")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("company")
	cmd.MarkFlagRequired("domain")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateTenantCmd) updateTenant(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("company"); err == nil && v != "" {
		body["company"] = v
	}
	if v, err := cmd.Flags().GetString("domain"); err == nil && v != "" {
		body["domain"] = v
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
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/tenants/{id}", pathParameters)

	return n.doUpdateTenant("PUT", path, queryValue, body)
}

func (n *updateTenantCmd) doUpdateTenant(method string, path string, query string, body map[string]interface{}) error {
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
