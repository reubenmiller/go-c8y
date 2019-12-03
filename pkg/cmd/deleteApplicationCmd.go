// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type deleteApplicationCmd struct {
	*baseCmd
}

func newDeleteApplicationCmd() *deleteApplicationCmd {
	ccmd := &deleteApplicationCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete application",
		Long:  `Info: The application can only be removed when its availability is PRIVATE or in other case when it has no subscriptions.`,
		Example: `
$ c8y applications delete --application 12345
Delete an application by id

$ c8y applications delete --application my-temp-app
Delete an application by name
		`,
		RunE: ccmd.deleteApplication,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("application", "", "Application id (required)")

	// Required flags
	cmd.MarkFlagRequired("application")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteApplicationCmd) deleteApplication(cmd *cobra.Command, args []string) error {

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

	// form data
	formData := make(map[string]io.Reader)

	// body
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)
	if cmd.Flags().Changed("application") {
		applicationInputValues, applicationValue, err := getApplicationSlice(cmd, args, "application")

		if err != nil {
			return newUserError("no matching applications found", applicationInputValues, err)
		}

		if len(applicationValue) == 0 {
			return newUserError("no matching applications found", applicationInputValues)
		}

		for _, item := range applicationValue {
			if item != "" {
				pathParameters["application"] = newIDValue(item).GetID()
			}
		}
	}

	path := replacePathParameters("/application/applications/{application}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	req := c8y.RequestOptions{
		Method:       "DELETE",
		Path:         path,
		Query:        queryValue,
		Body:         body.GetMap(),
		FormData:     formData,
		IgnoreAccept: false,
		DryRun:       globalFlagDryRun,
	}

	return n.doDeleteApplication(req, filters)
}

func (n *deleteApplicationCmd) doDeleteApplication(req c8y.RequestOptions, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		req,
	)

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "")
		} else {
			responseText = []byte(*resp.JSONData)
		}

		if globalFlagPrettyPrint && json.Valid(responseText) {
			fmt.Printf("%s", pretty.Pretty(responseText))
		} else {
			fmt.Printf("%s", responseText)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
