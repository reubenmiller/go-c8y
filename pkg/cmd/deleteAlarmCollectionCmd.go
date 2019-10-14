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

type deleteAlarmCollectionCmd struct {
	*baseCmd
}

func newDeleteAlarmCollectionCmd() *deleteAlarmCollectionCmd {
	ccmd := &deleteAlarmCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "deleteCollection",
		Short: "Delete a collection of alarms",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteAlarmCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Source device id.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of alarm occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of alarm occurrence.")
	cmd.Flags().String("type", "", "Alarm type.")
	cmd.Flags().String("fragmentType", "", "Alarm fragment type.")
	cmd.Flags().String("severity", "", "Alarm severity, for example CRITICAL, MAJOR, MINOR or WARNING.")
	cmd.Flags().Bool("resolved", false, "When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.")
	cmd.Flags().Bool("withSourceAssets", false, "When set to true also alarms for related source assets will be removed. When this parameter is provided also source must be defined.")
	cmd.Flags().Bool("withSourceDevices", false, "When set to true also alarms for related source devices will be removed. When this parameter is provided also source must be defined.")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteAlarmCollectionCmd) deleteAlarmCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	deviceValue := getFormattedDeviceSlice(cmd, args, "device")
	if len(deviceValue) > 0 {
		for _, item := range deviceValue {
			if item != "" {
				query.Add("source", newIDValue(item).GetID())
			}
		}
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
	if v, err := cmd.Flags().GetString("status"); err == nil {
		if v != "" {
			query.Add("status", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil {
		if v != "" {
			query.Add("severity", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("resolved"); err == nil {
		if v {
			query.Add("resolved", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("withSourceAssets"); err == nil {
		if v {
			query.Add("withSourceAssets", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("withSourceDevices"); err == nil {
		if v {
			query.Add("withSourceDevices", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
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

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("alarm/alarms", pathParameters)

	return n.doDeleteAlarmCollection("DELETE", path, queryValue, body)
}

func (n *deleteAlarmCollectionCmd) doDeleteAlarmCollection(method string, path string, query string, body map[string]interface{}) error {
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
