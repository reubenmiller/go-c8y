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

type deleteMeasurementCmd struct {
	*baseCmd
}

func newDeleteMeasurementCmd() *deleteMeasurementCmd {
	ccmd := &deleteMeasurementCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete measurement/s",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteMeasurement,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Measurement id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteMeasurementCmd) deleteMeasurement(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("measurement/measurements/{id}", pathParameters)

	return n.doDeleteMeasurement("DELETE", path, queryValue, body)
}

func (n *deleteMeasurementCmd) doDeleteMeasurement(method string, path string, query string, body map[string]interface{}) error {
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
