package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getAlarmCollectionCmd struct {
	*baseCmd
}

func newGetAlarmCollectionCmd() *getAlarmCollectionCmd {
	ccmd := &getAlarmCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a collection of alarms based on filter parameters",
		Long:  `Get a collection of alarms based on filter parameters`,
		Example: `
        c8y alarms get --type value --severity MAJOR
		`,
		RunE: ccmd.getAlarmCollection,
	}

	cmd.Flags().String("source", "", "Source device id.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of alarm occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of alarm occurrence.")
	cmd.Flags().String("type", "", "Alarm type.")
	cmd.Flags().String("fragmentType", "", "")
	cmd.Flags().String("status", "", "Comma separated alarm statuses, for example ACTIVE,CLEARED.")
	cmd.Flags().String("severity", "", "Alarm severity, for example MINOR.")
	cmd.Flags().Bool("resolved", false, "When set to true only resolved alarms will be removed (the one with status CLEARED), false means alarms with status ACTIVE or ACKNOWLEDGED.")
	cmd.Flags().Bool("withAssets", false, "Include assets")
	cmd.Flags().Bool("withDevices", false, "Include devices")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getAlarmCollectionCmd) getAlarmCollection(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetBool("withAssets"); err == nil {
		if v {
			query.Add("withAssets", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetBool("withDevices"); err == nil {
		if v {
			query.Add("withDevices", "true")
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

	path := replacePathParameters("alarm/alarms", pathParameters)

	return n.doGetAlarmCollection("GET", path, queryValue, body)
}

func (n *getAlarmCollectionCmd) doGetAlarmCollection(method string, path string, query string, body map[string]interface{}) error {
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
