package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getTenantUsageStatisticsCollectionCmd struct {
	*baseCmd
}

func newGetTenantUsageStatisticsCollectionCmd() *getTenantUsageStatisticsCollectionCmd {
	ccmd := &getTenantUsageStatisticsCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get collection of tenant usage statistics",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getTenantUsageStatisticsCollection,
	}

	cmd.Flags().String("dateFrom", "", "Start date or date and time of the statistics.")
	cmd.Flags().String("dateTo", "", "End date or date and time of the statistics.")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getTenantUsageStatisticsCollectionCmd) getTenantUsageStatisticsCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
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
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/tenant/statistics", pathParameters)

	return n.doGetTenantUsageStatisticsCollection("GET", path, queryValue, body)
}

func (n *getTenantUsageStatisticsCollectionCmd) doGetTenantUsageStatisticsCollection(method string, path string, query string, body map[string]interface{}) error {
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
