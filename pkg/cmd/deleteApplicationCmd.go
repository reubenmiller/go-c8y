package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteApplicationCmd struct {
	*baseCmd
}

func newDeleteApplicationCmd() *deleteApplicationCmd {
	ccmd := &deleteApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete application",
		Long:  `Info: The application can only be removed when its availability is PRIVATE or in other case when it has no subscriptions.`,
		Example: `
        
		`,
		RunE: ccmd.deleteApplication,
	}

	cmd.Flags().String("id", "", "Application id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteApplicationCmd) deleteApplication(cmd *cobra.Command, args []string) error {

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

	return n.doDeleteApplication("DELETE", path, queryValue, body)
}

func (n *deleteApplicationCmd) doDeleteApplication(method string, path string, query string, body map[string]interface{}) error {
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
