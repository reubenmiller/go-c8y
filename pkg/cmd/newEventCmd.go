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

type newEventCmd struct {
	*baseCmd
}

func newNewEventCmd() *newEventCmd {
	ccmd := &newEventCmd{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create event",
		Long:  `Create event`,
		Example: `
        
		`,
		RunE: ccmd.newEvent,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject which is the source of this event. (required)")
	cmd.Flags().String("time", "", "Time of the event. (required)")
	cmd.Flags().String("type", "", "Identifies the type of this event. (required)")
	cmd.Flags().String("text", "", "Text description of the event. (required)")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("time")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("text")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newEventCmd) newEvent(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("device"); err == nil && v != "" {
		if _, exists := body["device"]; !exists {
			body["source"] = make(map[string]interface{})
		}
		body["source"].(map[string]interface{})["id"] = v
	}
	if v, err := cmd.Flags().GetString("time"); err == nil && v != "" {
		body["time"] = v
	}
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil && v != "" {
		body["text"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("event/events", pathParameters)

	return n.doNewEvent("POST", path, queryValue, body)
}

func (n *newEventCmd) doNewEvent(method string, path string, query string, body map[string]interface{}) error {
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
