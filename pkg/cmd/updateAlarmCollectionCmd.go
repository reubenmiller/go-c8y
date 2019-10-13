package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateAlarmCollectionCmd struct {
	*baseCmd
}

func newUpdateAlarmCollectionCmd() *updateAlarmCollectionCmd {
	ccmd := &updateAlarmCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "The PUT method allows for updating alarms collections. Currently only the status of alarms can be changed",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateAlarmCollection,
	}

	cmd.Flags().String("source", "", "The ManagedObject that the alarm originated from")
	cmd.Flags().String("status", "", "The status of the alarm: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new alarm will have status ACTIVE. Must be upper-case.")
	cmd.Flags().String("severity", "", "The severity of the alarm: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case.")
	cmd.Flags().Bool("resolved", false, "When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of alarm occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of alarm occurrence.")
	cmd.Flags().String("newStatus", "", "New status to be applied to all of the matching alarms")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateAlarmCollectionCmd) updateAlarmCollection(cmd *cobra.Command, args []string) error {

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
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("newStatus"); err == nil && v != "" {
		body["status"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("alarm/alarms", pathParameters)

	return n.doUpdateAlarmCollection("PUT", path, queryValue, body)
}

func (n *updateAlarmCollectionCmd) doUpdateAlarmCollection(method string, path string, query string, body map[string]interface{}) error {
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
