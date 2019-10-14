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

type disableApplicationFromTenantCmd struct {
	*baseCmd
}

func newDisableApplicationFromTenantCmd() *disableApplicationFromTenantCmd {
	ccmd := &disableApplicationFromTenantCmd{}

	cmd := &cobra.Command{
		Use:   "disableApplication",
		Short: "Disable application on tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.disableApplicationFromTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant id")
	cmd.Flags().String("application", "", "Application id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *disableApplicationFromTenantCmd) disableApplicationFromTenant(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("application"); err == nil {
		pathParameters["application"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/tenants/{tenant}/applications/{application}", pathParameters)

	return n.doDisableApplicationFromTenant("DELETE", path, queryValue, body)
}

func (n *disableApplicationFromTenantCmd) doDisableApplicationFromTenant(method string, path string, query string, body map[string]interface{}) error {
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
