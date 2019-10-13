package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
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

	cmd.Flags().String("category", "", "Category of option")
	cmd.Flags().String("key", "", "Key of option")
	cmd.Flags().String("value", "", "Value of option")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newTenantOptionCmd) newTenantOption(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

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
