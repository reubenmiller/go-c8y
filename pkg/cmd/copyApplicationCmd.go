package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type copyApplicationCmd struct {
	*baseCmd
}

func newCopyApplicationCmd() *copyApplicationCmd {
	ccmd := &copyApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy application",
		Long: `A POST request to the 'clone' resource creates a new application based on an already existing one.
The properties are copied to the newly created application. For name, key and context path a 'clone' prefix is added in order to be unique.
If the target application is hosted and has an active version, the new application will have the active version with the same content.
The response contains a representation of the newly created application.
Required role ROLE_APPLICATION_MANAGMENT_ADMIN
`,
		Example: `
        
		`,
		RunE: ccmd.copyApplication,
	}

	cmd.Flags().String("id", "", "Application id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *copyApplicationCmd) copyApplication(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/application/applications/{id}/clone", pathParameters)

	return n.doCopyApplication("POST", path, queryValue, body)
}

func (n *copyApplicationCmd) doCopyApplication(method string, path string, query string, body map[string]interface{}) error {
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
