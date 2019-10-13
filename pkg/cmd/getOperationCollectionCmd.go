package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getOperationCollectionCmd struct {
	*baseCmd
}

func newGetOperationCollectionCmd() *getOperationCollectionCmd {
	ccmd := &getOperationCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getCollection",
		Short: "Get a collection of operations based on filter parameters",
		Long:  `Get a collection of operations based on filter parameters`,
		Example: `
        Get a list of pending operations
c8y operation get --status PENDING
		`,
		RunE: ccmd.getOperationCollection,
	}

	cmd.Flags().String("agentId", "", "Agent ID")
	cmd.Flags().String("deviceId", "", "Device ID")
	cmd.Flags().String("dateFrom", "", "")
	cmd.Flags().String("dateTo", "", "")
	cmd.Flags().String("status", "", "Operation status, can be one of SUCCESSFUL, FAILED, EXECUTING or PENDING.")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getOperationCollectionCmd) getOperationCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("agentId"); err == nil {
		if v != "" {
			query.Add("agentId", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("deviceId"); err == nil {
		if v != "" {
			query.Add("deviceId", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateFrom"); err == nil {
		if v != "" {
			query.Add("dateFrom", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("dateTo"); err == nil {
		if v != "" {
			query.Add("dateTo", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("status"); err == nil {
		if v != "" {
			query.Add("status", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("devicecontrol/operations", pathParameters)

	return n.doGetOperationCollection("GET", path, queryValue, body)
}

func (n *getOperationCollectionCmd) doGetOperationCollection(method string, path string, query string, body map[string]interface{}) error {
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
