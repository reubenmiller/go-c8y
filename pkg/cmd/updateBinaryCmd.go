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

type updateBinaryCmd struct {
	*baseCmd
}

func newUpdateBinaryCmd() *updateBinaryCmd {
	ccmd := &updateBinaryCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update inventory binary",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateBinary,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Inventory binary id (required)")

	// Required flags
	cmd.MarkFlagRequired("id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateBinaryCmd) updateBinary(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("file"); err == nil && v != "" {
		body["file"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/inventory/binaries/{id}", pathParameters)

	return n.doUpdateBinary("PUT", path, queryValue, body)
}

func (n *updateBinaryCmd) doUpdateBinary(method string, path string, query string, body map[string]interface{}) error {
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
