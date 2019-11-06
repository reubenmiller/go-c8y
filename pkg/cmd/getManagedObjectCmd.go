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

type getManagedObjectCmd struct {
	*baseCmd
}

func newGetManagedObjectCmd() *getManagedObjectCmd {
	ccmd := &getManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get inventory/s",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getManagedObject,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "ManagedObject id (required)")
	cmd.Flags().Bool("withParents", false, "include a flat list of all parents and grandparents of the given object")

	// Required flags
	cmd.MarkFlagRequired("id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getManagedObjectCmd) getManagedObject(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetBool("withParents"); err == nil {
		if v {
			query.Add("withParents", "true")
		}
	} else {
		return newUserError("Flag does not exist")
	}
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

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "id", err))
	}

	path := replacePathParameters("inventory/managedObjects/{id}", pathParameters)

	return n.doGetManagedObject("GET", path, queryValue, body)
}

func (n *getManagedObjectCmd) doGetManagedObject(method string, path string, query string, body map[string]interface{}) error {
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
