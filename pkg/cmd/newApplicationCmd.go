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

type newApplicationCmd struct {
	*baseCmd
}

func newNewApplicationCmd() *newApplicationCmd {
	ccmd := &newApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "New application",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newApplication,
	}

	cmd.SilenceUsage = true

	addDataFlag(cmd)
	cmd.Flags().String("name", "", "Name of application")
	cmd.Flags().String("key", "", "Shared secret of application")
	cmd.Flags().String("type", "", "Type of application. Possible values are EXTERNAL, HOSTED, MICROSERVICE")
	cmd.Flags().String("availability", "", "Application will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *].")
	cmd.Flags().String("contextPath", "", "contextPath of the hosted application. Required when application type is HOSTED")
	cmd.Flags().String("resourcesUrl", "", "URL to application base directory hosted on an external server. Required when application type is HOSTED")
	cmd.Flags().String("resourcesUsername", "", "authorization username to access resourcesUrl")
	cmd.Flags().String("resourcesPassword", "", "authorization password to access resourcesUrl")
	cmd.Flags().String("externalUrl", "", "URL to the external application. Required when application type is EXTERNAL")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newApplicationCmd) newApplication(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("name"); err == nil && v != "" {
		body["name"] = v
	}
	if v, err := cmd.Flags().GetString("key"); err == nil && v != "" {
		body["key"] = v
	}
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("availability"); err == nil && v != "" {
		body["availability"] = v
	}
	if v, err := cmd.Flags().GetString("contextPath"); err == nil && v != "" {
		body["contextPath"] = v
	}
	if v, err := cmd.Flags().GetString("resourcesUrl"); err == nil && v != "" {
		body["resourcesUrl"] = v
	}
	if v, err := cmd.Flags().GetString("resourcesUsername"); err == nil && v != "" {
		body["resourcesUsername"] = v
	}
	if v, err := cmd.Flags().GetString("resourcesPassword"); err == nil && v != "" {
		body["resourcesPassword"] = v
	}
	if v, err := cmd.Flags().GetString("externalUrl"); err == nil && v != "" {
		body["externalUrl"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/application/applications", pathParameters)

	return n.doNewApplication("POST", path, queryValue, body)
}

func (n *newApplicationCmd) doNewApplication(method string, path string, query string, body map[string]interface{}) error {
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
