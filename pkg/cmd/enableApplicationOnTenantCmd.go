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

type enableApplicationOnTenantCmd struct {
	*baseCmd
}

func newEnableApplicationOnTenantCmd() *enableApplicationOnTenantCmd {
	ccmd := &enableApplicationOnTenantCmd{}

	cmd := &cobra.Command{
		Use:   "enableApplication",
		Short: "Enable application on tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.enableApplicationOnTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Tenant id")
	cmd.Flags().String("application", "", "Application id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *enableApplicationOnTenantCmd) enableApplicationOnTenant(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("application"); err == nil && v != "" {
		if _, exists := body["application"]; !exists {
			body["application"] = make(map[string]interface{})
		}
		body["application"].(map[string]interface{})["self"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/tenants/{id}/applications", pathParameters)

	return n.doEnableApplicationOnTenant("POST", path, queryValue, body)
}

func (n *enableApplicationOnTenantCmd) doEnableApplicationOnTenant(method string, path string, query string, body map[string]interface{}) error {
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
