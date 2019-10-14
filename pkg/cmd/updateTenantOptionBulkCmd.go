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

type updateTenantOptionBulkCmd struct {
	*baseCmd
}

func newUpdateTenantOptionBulkCmd() *updateTenantOptionBulkCmd {
	ccmd := &updateTenantOptionBulkCmd{}

	cmd := &cobra.Command{
		Use:   "updateBulk",
		Short: "Update multiple tenant options in provided category",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateTenantOptionBulk,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("category", "", "Tenant Option category")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateTenantOptionBulkCmd) updateTenantOptionBulk(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("category"); err == nil {
		pathParameters["category"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/options/{category}", pathParameters)

	return n.doUpdateTenantOptionBulk("PUT", path, queryValue, body)
}

func (n *updateTenantOptionBulkCmd) doUpdateTenantOptionBulk(method string, path string, query string, body map[string]interface{}) error {
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
