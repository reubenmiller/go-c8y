package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getTenantOptionsForCategoryCmd struct {
	*baseCmd
}

func newGetTenantOptionsForCategoryCmd() *getTenantOptionsForCategoryCmd {
	ccmd := &getTenantOptionsForCategoryCmd{}

	cmd := &cobra.Command{
		Use:   "getForCategory",
		Short: "Get tenant options for category",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getTenantOptionsForCategory,
	}

	cmd.Flags().String("category", "", "Tenant Option category")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getTenantOptionsForCategoryCmd) getTenantOptionsForCategory(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/tenant/options/{category}", pathParameters)

	return n.doGetTenantOptionsForCategory("GET", path, queryValue, body)
}

func (n *getTenantOptionsForCategoryCmd) doGetTenantOptionsForCategory(method string, path string, query string, body map[string]interface{}) error {
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
