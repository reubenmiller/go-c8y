package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateEventBinaryCmd struct {
	*baseCmd
}

func newUpdateEventBinaryCmd() *updateEventBinaryCmd {
	ccmd := &updateEventBinaryCmd{}

	cmd := &cobra.Command{
		Use:   "updateBinary",
		Short: "Update event binary",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.updateEventBinary,
	}

	cmd.Flags().String("id", "", "Event id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateEventBinaryCmd) updateEventBinary(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("file"); err == nil && v != "" {
		body["file"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("event/events/{id}/binaries", pathParameters)

	return n.doUpdateEventBinary("PUT", path, queryValue, body)
}

func (n *updateEventBinaryCmd) doUpdateEventBinary(method string, path string, query string, body map[string]interface{}) error {
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
