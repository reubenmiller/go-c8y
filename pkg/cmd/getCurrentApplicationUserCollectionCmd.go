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

type getCurrentApplicationUserCollectionCmd struct {
	*baseCmd
}

func newGetCurrentApplicationUserCollectionCmd() *getCurrentApplicationUserCollectionCmd {
	ccmd := &getCurrentApplicationUserCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "listSubscriptions",
		Short: "Get current application subscriptions",
		Long:  `Required authentication with bootstrap user`,
		Example: `
        
		`,
		RunE: ccmd.getCurrentApplicationUserCollection,
	}

	cmd.SilenceUsage = true

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getCurrentApplicationUserCollectionCmd) getCurrentApplicationUserCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/application/currentApplication/subscriptions", pathParameters)

	return n.doGetCurrentApplicationUserCollection("GET", path, queryValue, body)
}

func (n *getCurrentApplicationUserCollectionCmd) doGetCurrentApplicationUserCollection(method string, path string, query string, body map[string]interface{}) error {
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
