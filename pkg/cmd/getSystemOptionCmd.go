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

type getSystemOptionCmd struct {
	*baseCmd
}

func newGetSystemOptionCmd() *getSystemOptionCmd {
	ccmd := &getSystemOptionCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get system option",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getSystemOption,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("category", "", "System Option category")
	cmd.Flags().String("key", "", "System Option key")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getSystemOptionCmd) getSystemOption(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("category"); err == nil {
		pathParameters["category"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("key"); err == nil {
		pathParameters["key"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/tenant/system/options/{category}/{key}", pathParameters)

	return n.doGetSystemOption("GET", path, queryValue, body)
}

func (n *getSystemOptionCmd) doGetSystemOption(method string, path string, query string, body map[string]interface{}) error {
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
