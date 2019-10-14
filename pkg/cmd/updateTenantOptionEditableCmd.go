// Code generated from specification version 1.0.0: DO NOT EDIT
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

type updateTenantOptionEditableCmd struct {
	*baseCmd
}

func newUpdateTenantOptionEditableCmd() *updateTenantOptionEditableCmd {
	ccmd := &updateTenantOptionEditableCmd{}

	cmd := &cobra.Command{
		Use:   "updateEdit",
		Short: "Update tenant option editibility",
		Long: `Required role:: ROLE_OPTION_MANAGEMENT_ADMIN, Required tenant management Example Request:: Update access.control.allow.origin option.
`,
		Example: `
        
		`,
		RunE: ccmd.updateTenantOptionEditable,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("category", "", "Tenant Option category (required)")
	cmd.Flags().String("key", "", "Tenant Option key (required)")
	cmd.Flags().String("editable", "", "Whether the tenant option should be editable or not (required)")

	// Required flags
	cmd.MarkFlagRequired("category")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("editable")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateTenantOptionEditableCmd) updateTenantOptionEditable(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("editable"); err == nil && v != "" {
		body["editable"] = v
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

	path := replacePathParameters("/tenant/options/{category}/{key}/editable", pathParameters)

	return n.doUpdateTenantOptionEditable("PUT", path, queryValue, body)
}

func (n *updateTenantOptionEditableCmd) doUpdateTenantOptionEditable(method string, path string, query string, body map[string]interface{}) error {
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
