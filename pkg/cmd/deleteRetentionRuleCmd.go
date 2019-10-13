package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteRetentionRuleCmd struct {
	*baseCmd
}

func newDeleteRetentionRuleCmd() *deleteRetentionRuleCmd {
	ccmd := &deleteRetentionRuleCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete retention rule",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.deleteRetentionRule,
	}

	cmd.Flags().String("id", "", "Retention rule id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteRetentionRuleCmd) deleteRetentionRule(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/retention/retentions/{id}", pathParameters)

	return n.doDeleteRetentionRule("DELETE", path, queryValue, body)
}

func (n *deleteRetentionRuleCmd) doDeleteRetentionRule(method string, path string, query string, body map[string]interface{}) error {
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
