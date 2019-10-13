package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getMeasurementSeriesCmd struct {
	*baseCmd
}

func newGetMeasurementSeriesCmd() *getMeasurementSeriesCmd {
	ccmd := &getMeasurementSeriesCmd{}

	cmd := &cobra.Command{
		Use:   "getSeries",
		Short: "Get a collection of measurements based on filter parameters",
		Long:  `Get a collection of measurements based on filter parameters`,
		Example: `
        Get a list of measurements
c8y measurement get

Get a list of series [nx_WEA_29_Delta.MDL10FG001] and [nx_WEA_29_Delta.ST9] for device 12345
measurement getSeries --source 12345 --series nx_WEA_29_Delta.MDL10FG001 --series nx_WEA_29_Delta.ST9 --dateFrom (Get-C8yDate (last 10min)) --dateTo (Get-C8yDate (last 0min))
		`,
		RunE: ccmd.getMeasurementSeries,
	}

	cmd.Flags().String("source", "", "Device ID")
	cmd.Flags().StringArray("series", []string{""}, "measurement type and series name, e.g. c8y_AccelerationMeasurement.acceleration")
	cmd.Flags().String("aggregationType", "", "Fragment name from measurement.")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of measurement occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of measurement occurrence.")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getMeasurementSeriesCmd) getMeasurementSeries(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetStringArray("series"); err == nil {
		if len(v) > 0 {
			for _, item := range v {
				if item != "" {
					query.Add("series", item)
				}
			}
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("aggregationType"); err == nil {
		if v != "" {
			query.Add("aggregationType", url.QueryEscape(v))
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

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("measurement/measurements/series", pathParameters)

	return n.doGetMeasurementSeries("GET", path, queryValue, body)
}

func (n *getMeasurementSeriesCmd) doGetMeasurementSeries(method string, path string, query string, body map[string]interface{}) error {
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
