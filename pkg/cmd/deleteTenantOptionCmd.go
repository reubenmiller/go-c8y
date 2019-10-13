package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteTenantOptionCmd struct {
	*baseCmd
}

func newDeleteTenantOptionCmd() *deleteTenantOptionCmd {
	ccmd := &deleteTenantOptionCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete tenant option",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteTenantOption,
	}

	cmd.Flags().String("category", "", "Tenant Option category")
	cmd.Flags().String("key", "", "Tenant Option key")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteTenantOptionCmd) deleteTenantOption(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

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

	return n.doDeleteTenantOption("DELETE", path, queryValue, body)
}

func (n *deleteTenantOptionCmd) doDeleteTenantOption(method string, path string, query string, body map[string]interface{}) error {
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
