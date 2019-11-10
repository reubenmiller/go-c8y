// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type disableApplicationFromTenantCmd struct {
	*baseCmd
}

func newDisableApplicationFromTenantCmd() *disableApplicationFromTenantCmd {
	ccmd := &disableApplicationFromTenantCmd{}

	cmd := &cobra.Command{
		Use:   "disableApplication",
		Short: "Disable application on tenant",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.disableApplicationFromTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant id")
	cmd.Flags().String("application", "", "Application id (required)")

	// Required flags
	cmd.MarkFlagRequired("application")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *disableApplicationFromTenantCmd) disableApplicationFromTenant(cmd *cobra.Command, args []string) error {

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
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)
	if v := getTenantWithDefaultFlag(cmd, "tenant", client.TenantName); v != "" {
		pathParameters["tenant"] = v
	}
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

	path := replacePathParameters("/tenant/tenants/{tenant}/applications/{application}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doDisableApplicationFromTenant("DELETE", path, queryValue, body.GetMap(), filters)
}

func (n *disableApplicationFromTenantCmd) doDisableApplicationFromTenant(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
			DryRun:       globalFlagDryRun,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {

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
