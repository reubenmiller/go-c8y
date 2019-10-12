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
		Short: "",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.updateAlarmCollection,
	}

	cmd.Flags().String("source", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("severity", "", "")
	cmd.Flags().Bool("resolved", false, "")
	cmd.Flags().String("dateFrom", "", "")
	cmd.Flags().String("dateTo", "", "")
	cmd.Flags().String("newStatus", "", "")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateAlarmCollectionCmd) updateAlarmCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("source"); err == nil {
		query.Add("source", url.QueryEscape(v))
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("status"); err == nil {
		query.Add("status", url.QueryEscape(v))
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil {
		query.Add("severity", url.QueryEscape(v))
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
		query.Add("dateFrom", url.QueryEscape(v))
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateTo"); err == nil {
		query.Add("dateTo", url.QueryEscape(v))
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
	if v, err := cmd.Flags().GetString("status"); err == nil {
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

	if resp.JSONData != nil {
		fmt.Println(*resp.JSONData)
	}
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
