package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getRetentionRuleCollectionCmd struct {
	*baseCmd
}

func newGetRetentionRuleCollectionCmd() *getRetentionRuleCollectionCmd {
	ccmd := &getRetentionRuleCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getRetentionRuleCollection",
		Short: "Get collection of retention rules",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getRetentionRuleCollection,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getRetentionRuleCollectionCmd) getRetentionRuleCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/retention/retentions", pathParameters)

	return n.doGetRetentionRuleCollection("GET", path, queryValue, body)
}

func (n *getRetentionRuleCollectionCmd) doGetRetentionRuleCollection(method string, path string, query string, body map[string]interface{}) error {
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
