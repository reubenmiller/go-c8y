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

type newAlarmCmd struct {
	*baseCmd
}

func newNewAlarmCmd() *newAlarmCmd {
	ccmd := &newAlarmCmd{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new alarm",
		Long:  `Create a new alarm`,
		Example: `
        
		`,
		RunE: ccmd.newAlarm,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("source", "", "The ManagedObject that the alarm originated from")
	cmd.Flags().String("type", "", "Identifies the type of this alarm, e.g. 'com_cumulocity_events_TamperEvent'.")
	cmd.Flags().String("time", "", "Time of the alarm.")
	cmd.Flags().String("text", "", "Text description of the alarm.")
	cmd.Flags().String("severity", "", "The severity of the alarm: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case.")
	cmd.Flags().String("status", "", "The status of the alarm: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new alarm will have status ACTIVE. Must be upper-case.")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newAlarmCmd) newAlarm(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("source"); err == nil && v != "" {
		if _, exists := body["source"]; !exists {
			body["source"] = make(map[string]interface{})
		}
		body["source"].(map[string]interface{})["id"] = v
	}
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("time"); err == nil && v != "" {
		body["time"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil && v != "" {
		body["text"] = v
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil && v != "" {
		body["severity"] = v
	}
	if v, err := cmd.Flags().GetString("status"); err == nil && v != "" {
		body["status"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("alarm/alarms", pathParameters)

	return n.doNewAlarm("POST", path, queryValue, body)
}

func (n *newAlarmCmd) doNewAlarm(method string, path string, query string, body map[string]interface{}) error {
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
