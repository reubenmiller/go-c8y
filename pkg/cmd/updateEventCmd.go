package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateEventCmd struct {
	*baseCmd
}

func newUpdateEventCmd() *updateEventCmd {
	ccmd := &updateEventCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an event",
		Long:  "Update an event",
		Example: `
        
		`,
		RunE: ccmd.updateEvent,
	}

	cmd.Flags().String("id", "", "Event id")
	cmd.Flags().String("text", "", "Text description of the event.")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateEventCmd) updateEvent(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("text"); err == nil && v != "" {
		body["text"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("event/events/{id}", pathParameters)

	return n.doUpdateEvent("PUT", path, queryValue, body)
}

func (n *updateEventCmd) doUpdateEvent(method string, path string, query string, body map[string]interface{}) error {
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
