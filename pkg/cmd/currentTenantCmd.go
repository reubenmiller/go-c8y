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

type currentTenantCmd struct {
	*baseCmd
}

func newCurrentTenantCmd() *currentTenantCmd {
	ccmd := &currentTenantCmd{}

	cmd := &cobra.Command{
		Use:   "getCurrentTenant",
		Short: "Get current tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.currentTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Tenant id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *currentTenantCmd) currentTenant(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/currentTenant", pathParameters)

	return n.doCurrentTenant("GET", path, queryValue, body)
}

func (n *currentTenantCmd) doCurrentTenant(method string, path string, query string, body map[string]interface{}) error {
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
