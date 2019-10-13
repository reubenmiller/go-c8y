package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteEventCollectionCmd struct {
	*baseCmd
}

func newDeleteEventCollectionCmd() *deleteEventCollectionCmd {
	ccmd := &deleteEventCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "deleteCollection",
		Short: "Delete a collection of events",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.deleteEventCollection,
	}

	cmd.Flags().String("source", "", "Device ID")
	cmd.Flags().String("type", "", "Event type.")
	cmd.Flags().String("fragmentType", "", "Fragment name from event.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of event occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of event occurrence.")
	cmd.Flags().Bool("revert", false, "Return the newest instead of the oldest events. Must be used with dateFrom and dateTo parameters")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteEventCollectionCmd) deleteEventCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("source"); err == nil {
		if v != "" {
			query.Add("source", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			query.Add("type", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("fragmentType"); err == nil {
		if v != "" {
			query.Add("fragmentType", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateFrom"); err == nil {
		if v != "" {
			query.Add("dateFrom", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateTo"); err == nil {
		if v != "" {
			query.Add("dateTo", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("revert"); err == nil {
		if v {
			query.Add("revert", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("event/events", pathParameters)

	return n.doDeleteEventCollection("DELETE", path, queryValue, body)
}

func (n *deleteEventCollectionCmd) doDeleteEventCollection(method string, path string, query string, body map[string]interface{}) error {
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
