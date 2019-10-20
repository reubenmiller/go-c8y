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

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject that the alarm originated from (required)")
	cmd.Flags().String("type", "", "Identifies the type of this alarm, e.g. 'com_cumulocity_events_TamperEvent'. (required)")
	cmd.Flags().String("time", "", "Time of the alarm. (required)")
	cmd.Flags().String("text", "", "Text description of the alarm. (required)")
	cmd.Flags().String("severity", "", "The severity of the alarm: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case. (required)")
	cmd.Flags().String("status", "", "The status of the alarm: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new alarm will have status ACTIVE. Must be upper-case.")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("time")
	cmd.MarkFlagRequired("text")
	cmd.MarkFlagRequired("severity")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newAlarmCmd) newAlarm(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if cmd.Flags().Changed("pageSize") {
		if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
			query.Add("pageSize", fmt.Sprintf("%d", v))
		}
	}

	if cmd.Flags().Changed("withTotalPages") {
		if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
			query.Add("withTotalPages", "true")
		}
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("device"); err == nil && v != "" {
		if _, exists := body["device"]; !exists {
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
		if globalFlagPrettyPrint {
			fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
		} else {
			fmt.Printf("%s\n", *resp.JSONData)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
