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

type getApplicationCollectionCmd struct {
	*baseCmd
}

func newGetApplicationCollectionCmd() *getApplicationCollectionCmd {
	ccmd := &getApplicationCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get collection of applications",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getApplicationCollection,
	}

	cmd.SilenceUsage = true

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getApplicationCollectionCmd) getApplicationCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/application/applications", pathParameters)

	return n.doGetApplicationCollection("GET", path, queryValue, body)
}

func (n *getApplicationCollectionCmd) doGetApplicationCollection(method string, path string, query string, body map[string]interface{}) error {
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
