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

type getExternalIDCmd struct {
	*baseCmd
}

func newGetExternalIDCmd() *getExternalIDCmd {
	ccmd := &getExternalIDCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get external id",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getExternalID,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("type", "", "External identity type (required)")
	cmd.Flags().String("name", "", "External identity id/name (required)")

	// Required flags
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("name")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getExternalIDCmd) getExternalID(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("type"); err == nil {
		pathParameters["type"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("name"); err == nil {
		pathParameters["name"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/identity/externalIds/{type}/{name}", pathParameters)

	return n.doGetExternalID("GET", path, queryValue, body)
}

func (n *getExternalIDCmd) doGetExternalID(method string, path string, query string, body map[string]interface{}) error {
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
