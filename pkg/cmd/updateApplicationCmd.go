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

type updateApplicationCmd struct {
	*baseCmd
}

func newUpdateApplicationCmd() *updateApplicationCmd {
	ccmd := &updateApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "updateBinary",
		Short: "Update application",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateApplication,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Application id (required)")
	addDataFlag(cmd)
	cmd.Flags().String("name", "", "Name of application")
	cmd.Flags().String("key", "", "Shared secret of application")
	cmd.Flags().String("availability", "", "Application will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *].")
	cmd.Flags().String("contextPath", "", "contextPath of the hosted application")
	cmd.Flags().String("resourcesUrl", "", "URL to application base directory hosted on an external server")
	cmd.Flags().String("resourcesUsername", "", "authorization username to access resourcesUrl")
	cmd.Flags().String("resourcesPassword", "", "authorization password to access resourcesUrl")
	cmd.Flags().String("externalUrl", "", "URL to the external application")

	// Required flags
	cmd.MarkFlagRequired("id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateApplicationCmd) updateApplication(cmd *cobra.Command, args []string) error {

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
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "id", err))
	}

	path := replacePathParameters("/application/applications/{id}", pathParameters)

	return n.doUpdateApplication("PUT", path, queryValue, body)
}

func (n *updateApplicationCmd) doUpdateApplication(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
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
