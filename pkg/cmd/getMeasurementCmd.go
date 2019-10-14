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

type getMeasurementCmd struct {
	*baseCmd
}

func newGetMeasurementCmd() *getMeasurementCmd {
	ccmd := &getMeasurementCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get measurement/s",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getMeasurement,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Measurement id (required)")

	// Required flags
	cmd.MarkFlagRequired("id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getMeasurementCmd) getMeasurement(cmd *cobra.Command, args []string) error {

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

	return n.doGetMeasurement("GET", path, queryValue, body)
}

func (n *getMeasurementCmd) doGetMeasurement(method string, path string, query string, body map[string]interface{}) error {
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
