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

type getExternalIDCollectionCmd struct {
	*baseCmd
}

func newGetExternalIDCollectionCmd() *getExternalIDCollectionCmd {
	ccmd := &getExternalIDCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a collection of external ids based on filter parameters",
		Long:  `Get a collection of external ids based on filter parameters`,
		Example: `
        Get a list of external ids
c8y identity list
		`,
		RunE: ccmd.getExternalIDCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device id")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getExternalIDCollectionCmd) getExternalIDCollection(cmd *cobra.Command, args []string) error {

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

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetStringSlice("device"); err == nil {
		for _, iValue := range v {
			pathParameters["device"] = iValue
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "device", err))
	}

	path := replacePathParameters("identity/globalIds/{device}/externalIds", pathParameters)

	return n.doGetExternalIDCollection("GET", path, queryValue, body)
}

func (n *getExternalIDCollectionCmd) doGetExternalIDCollection(method string, path string, query string, body map[string]interface{}) error {
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
