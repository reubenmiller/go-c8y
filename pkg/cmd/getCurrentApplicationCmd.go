package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getCurrentApplicationCmd struct {
	*baseCmd
}

func newGetCurrentApplicationCmd() *getCurrentApplicationCmd {
	ccmd := &getCurrentApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get current application",
		Long:  `Required authentication with bootstrap user`,
		Example: `
        
		`,
		RunE: ccmd.getCurrentApplication,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getCurrentApplicationCmd) getCurrentApplication(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/application/currentApplication", pathParameters)

	return n.doGetCurrentApplication("GET", path, queryValue, body)
}

func (n *getCurrentApplicationCmd) doGetCurrentApplication(method string, path string, query string, body map[string]interface{}) error {
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
