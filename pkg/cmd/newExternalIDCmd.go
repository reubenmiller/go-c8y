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

type newExternalIDCmd struct {
	*baseCmd
}

func newNewExternalIDCmd() *newExternalIDCmd {
	ccmd := &newExternalIDCmd{}

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new external id",
		Long:  `Create a new external id`,
		Example: `
        
		`,
		RunE: ccmd.newExternalID,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("deviceId", "", "The ManagedObject linked to the external ID.")
	cmd.Flags().String("type", "", "The type of the external identifier as string, e.g. 'com_cumulocity_model_idtype_SerialNumber'.")
	cmd.Flags().String("name", "", "The identifier used in the external system that Cumulocity interfaces with.")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newExternalIDCmd) newExternalID(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("name"); err == nil && v != "" {
		body["externalId"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("deviceId"); err == nil {
		pathParameters["deviceId"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("identity/globalIds/{deviceId}/externalIds", pathParameters)

	return n.doNewExternalID("POST", path, queryValue, body)
}

func (n *newExternalIDCmd) doNewExternalID(method string, path string, query string, body map[string]interface{}) error {
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
