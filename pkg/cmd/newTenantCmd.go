package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
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

	cmd.Flags().String("company", "", "Company name. Maximum 256 characters")
	cmd.Flags().String("domain", "", "Domain name to be used for the tenant. Maximum 256 characters")
	cmd.Flags().String("id", "", "The tenant ID. Will be auto-generated if not present.")
	cmd.Flags().String("adminName", "", "Username of the tenant administrator")
	cmd.Flags().String("adminPass", "", "Password of the tenant administrator")
	cmd.Flags().String("contactName", "", "A contact name, for example an administrator, of the tenant")
	cmd.Flags().String("contact_phone", "", "An international contact phone number")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newTenantCmd) newTenant(cmd *cobra.Command, args []string) error {

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
			Method: method,
			Path:   path,
			Query:  query,
			Body:   body,
		})

	if resp != nil && resp.JSONData != nil {
		fmt.Println(*resp.JSONData)
	}
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
