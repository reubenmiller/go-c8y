package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateAlarmCmd struct {
	*baseCmd
}

func newUpdateAlarmCmd() *updateAlarmCmd {
	ccmd := &updateAlarmCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.updateAlarm,
	}

	cmd.Flags().String("status", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().String("text", "", "")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateAlarmCmd) updateAlarm(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("status"); err == nil {
		body["status"] = v
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil {
		body["severity"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil {
		body["text"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("alarm/alarms/{id}", pathParameters)

	return n.doUpdateAlarm("PUT", path, queryValue, body)
}

func (n *updateAlarmCmd) doUpdateAlarm(method string, path string, query string, body map[string]interface{}) error {
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
