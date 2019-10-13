package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateTenantOptionCmd struct {
	*baseCmd
}

func newUpdateTenantOptionCmd() *updateTenantOptionCmd {
	ccmd := &updateTenantOptionCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update tenant option",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateTenantOption,
	}

	cmd.Flags().String("category", "", "Tenant Option category")
	cmd.Flags().String("key", "", "Tenant Option key")
	cmd.Flags().String("value", "", "New value")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateTenantOptionCmd) updateTenantOption(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("value"); err == nil && v != "" {
		body["value"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("category"); err == nil {
		pathParameters["category"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("key"); err == nil {
		pathParameters["key"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/options/{category}/{key}", pathParameters)

	return n.doUpdateTenantOption("PUT", path, queryValue, body)
}

func (n *updateTenantOptionCmd) doUpdateTenantOption(method string, path string, query string, body map[string]interface{}) error {
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
