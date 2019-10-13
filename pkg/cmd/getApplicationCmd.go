package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getApplicationCmd struct {
	*baseCmd
}

func newGetApplicationCmd() *getApplicationCmd {
	ccmd := &getApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get application",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getApplication,
	}

	cmd.Flags().String("id", "", "Application id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getApplicationCmd) getApplication(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/application/applications/{id}", pathParameters)

	return n.doGetApplication("GET", path, queryValue, body)
}

func (n *getApplicationCmd) doGetApplication(method string, path string, query string, body map[string]interface{}) error {
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
