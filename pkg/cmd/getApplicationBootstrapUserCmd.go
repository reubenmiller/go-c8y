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

type getApplicationBootstrapUserCmd struct {
	*baseCmd
}

func newGetApplicationBootstrapUserCmd() *getApplicationBootstrapUserCmd {
	ccmd := &getApplicationBootstrapUserCmd{}

	cmd := &cobra.Command{
		Use:   "getBootstrapUser",
		Short: "Get application bootstrap user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getApplicationBootstrapUser,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Application id (required)")

	// Required flags
	cmd.MarkFlagRequired("id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getApplicationBootstrapUserCmd) getApplicationBootstrapUser(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/application/applications/{id}/bootstrapUser", pathParameters)

	return n.doGetApplicationBootstrapUser("GET", path, queryValue, body)
}

func (n *getApplicationBootstrapUserCmd) doGetApplicationBootstrapUser(method string, path string, query string, body map[string]interface{}) error {
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
