package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateCurrentApplicationCmd struct {
	*baseCmd
}

func newUpdateCurrentApplicationCmd() *updateCurrentApplicationCmd {
	ccmd := &updateCurrentApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update current application",
		Long:  `Required authentication with bootstrap user`,
		Example: `
        
		`,
		RunE: ccmd.updateCurrentApplication,
	}

	cmd.Flags().String("id", "", "Application id")
	addDataFlag(cmd)
	cmd.Flags().String("name", "", "Name of application")
	cmd.Flags().String("key", "", "Shared secret of application")
	cmd.Flags().String("availability", "", "Application will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *].")
	cmd.Flags().String("contextPath", "", "contextPath of the hosted application")
	cmd.Flags().String("resourcesUrl", "", "URL to application base directory hosted on an external server")
	cmd.Flags().String("resourcesUsername", "", "authorization username to access resourcesUrl")
	cmd.Flags().String("resourcesPassword", "", "authorization password to access resourcesUrl")
	cmd.Flags().String("externalUrl", "", "URL to the external application")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateCurrentApplicationCmd) updateCurrentApplication(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/application/currentApplication", pathParameters)

	return n.doUpdateCurrentApplication("PUT", path, queryValue, body)
}

func (n *updateCurrentApplicationCmd) doUpdateCurrentApplication(method string, path string, query string, body map[string]interface{}) error {
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
