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

type queryManagedObjectCollectionCmd struct {
	*baseCmd
}

func newQueryManagedObjectCollectionCmd() *queryManagedObjectCollectionCmd {
	ccmd := &queryManagedObjectCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Get a collection of managedObjects based on Cumulocity query language",
		Long:  `Get a collection of managedObjects based on Cumulocity query language`,
		Example: `
        c8y managedObjects query --type value --severity MAJOR
		`,
		RunE: ccmd.queryManagedObjectCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("query", "", "ManagedObject query.")
	cmd.Flags().Bool("withParents", false, "include a flat list of all parents and grandparents of the given object")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *queryManagedObjectCollectionCmd) queryManagedObjectCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("query"); err == nil {
		if v != "" {
			query.Add("query", url.QueryEscape(v))
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "query", err))
	}
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
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("inventory/managedObjects", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	return n.doQueryManagedObjectCollection("GET", path, queryValue, body.GetMap(), filters)
}

func (n *queryManagedObjectCollectionCmd) doQueryManagedObjectCollection(method string, path string, query string, body map[string]interface{}, filters *JSONFilters) error {
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
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "managedObjects")
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
