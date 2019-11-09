// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
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

func (n *updateCurrentApplicationCmd) updateCurrentApplication(cmd *cobra.Command, args []string) error {

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
	body.SetMap(getDataFlag(cmd))
	if v, err := cmd.Flags().GetString("name"); err == nil {
		if v != "" {
			body.Set("name", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "name", err))
	}
	if v, err := cmd.Flags().GetString("key"); err == nil {
		if v != "" {
			body.Set("key", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "key", err))
	}
	if v, err := cmd.Flags().GetString("availability"); err == nil {
		if v != "" {
			body.Set("availability", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "availability", err))
	}
	if v, err := cmd.Flags().GetString("contextPath"); err == nil {
		if v != "" {
			body.Set("contextPath", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "contextPath", err))
	}
	if v, err := cmd.Flags().GetString("resourcesUrl"); err == nil {
		if v != "" {
			body.Set("resourcesUrl", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "resourcesUrl", err))
	}
	if v, err := cmd.Flags().GetString("resourcesUsername"); err == nil {
		if v != "" {
			body.Set("resourcesUsername", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "resourcesUsername", err))
	}
	if v, err := cmd.Flags().GetString("resourcesPassword"); err == nil {
		if v != "" {
			body.Set("resourcesPassword", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "resourcesPassword", err))
	}
	if v, err := cmd.Flags().GetString("externalUrl"); err == nil {
		if v != "" {
			body.Set("externalUrl", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "externalUrl", err))
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		if v != "" {
			pathParameters["id"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "id", err))
	}

	path := replacePathParameters("/application/currentApplication", pathParameters)

	return n.doUpdateCurrentApplication("PUT", path, queryValue, body.GetMap())
}

func (n *updateCurrentApplicationCmd) doUpdateCurrentApplication(method string, path string, query string, body map[string]interface{}) error {
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
