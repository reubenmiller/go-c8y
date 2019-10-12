package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type newAlarmCmd struct {
	*baseCmd
}

func newNewAlarmCmd() *newAlarmCmd {
	ccmd := &newAlarmCmd{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new alarm",
		Long:  "Create a new alarm",
		Example: `
        
		`,
		RunE: ccmd.newAlarm,
	}

	cmd.Flags().String("source", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("time", "", "")
	cmd.Flags().String("text", "", "")
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
	if v, err := cmd.Flags().GetString("type"); err == nil {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("time"); err == nil {
		body["time"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil {
		body["text"] = v
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil {
		body["severity"] = v
	}
	if v, err := cmd.Flags().GetString("status"); err == nil {
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

	if resp.JSONData != nil {
		fmt.Println(*resp.JSONData)
	}
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
