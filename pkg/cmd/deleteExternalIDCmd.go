package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteExternalIDCmd struct {
	*baseCmd
}

func newDeleteExternalIDCmd() *deleteExternalIDCmd {
	ccmd := &deleteExternalIDCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete external id",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.deleteExternalID,
	}

	cmd.Flags().String("type", "", "External identity type")
	cmd.Flags().String("name", "", "External identity id/name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteExternalIDCmd) deleteExternalID(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("type"); err == nil {
		pathParameters["type"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("name"); err == nil {
		pathParameters["name"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/identity/externalIds/{type}/{name}", pathParameters)

	return n.doDeleteExternalID("DELETE", path, queryValue, body)
}

func (n *deleteExternalIDCmd) doDeleteExternalID(method string, path string, query string, body map[string]interface{}) error {
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
