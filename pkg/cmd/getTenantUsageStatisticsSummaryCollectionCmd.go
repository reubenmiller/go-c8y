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

type getTenantUsageStatisticsSummaryCollectionCmd struct {
	*baseCmd
}

func newGetTenantUsageStatisticsSummaryCollectionCmd() *getTenantUsageStatisticsSummaryCollectionCmd {
	ccmd := &getTenantUsageStatisticsSummaryCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "listSummaryForTenant",
		Short: "Get collection of tenant usage statistics summary",
		Long:  `Get summary of requests and database usage from the start of this month until now`,
		Example: `
        
		`,
		RunE: ccmd.getTenantUsageStatisticsSummaryCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("dateFrom", "", "Start date or date and time of the statistics.")
	cmd.Flags().String("dateTo", "", "End date or date and time of the statistics.")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getTenantUsageStatisticsSummaryCollectionCmd) getTenantUsageStatisticsSummaryCollection(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/tenant/statistics/summary", pathParameters)

	return n.doGetTenantUsageStatisticsSummaryCollection("GET", path, queryValue, body)
}

func (n *getTenantUsageStatisticsSummaryCollectionCmd) doGetTenantUsageStatisticsSummaryCollection(method string, path string, query string, body map[string]interface{}) error {
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
