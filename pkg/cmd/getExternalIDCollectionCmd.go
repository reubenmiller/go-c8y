package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getExternalIDCollectionCmd struct {
	*baseCmd
}

func newGetExternalIDCollectionCmd() *getExternalIDCollectionCmd {
	ccmd := &getExternalIDCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getCollection",
		Short: "Get a collection of external ids based on filter parameters",
		Long:  "Get a collection of external ids based on filter parameters",
		Example: `
        Get a list of external ids
c8y identity getCollection
		`,
		RunE: ccmd.getExternalIDCollection,
	}

	cmd.Flags().String("deviceId", "", "Device id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getExternalIDCollectionCmd) getExternalIDCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("deviceId"); err == nil {
		pathParameters["deviceId"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("identity/globalIds/{deviceId}/externalIds", pathParameters)

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

	if resp != nil && resp.JSONData != nil {
		fmt.Println(*resp.JSONData)
	}
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
