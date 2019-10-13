package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type deleteAuditRecordCmd struct {
	*baseCmd
}

func newDeleteAuditRecordCmd() *deleteAuditRecordCmd {
	ccmd := &deleteAuditRecordCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an audit record",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.deleteAuditRecord,
	}

	cmd.Flags().String("id", "", "Audit record id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteAuditRecordCmd) deleteAuditRecord(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/audit/auditRecords/{id}", pathParameters)

	return n.doDeleteAuditRecord("DELETE", path, queryValue, body)
}

func (n *deleteAuditRecordCmd) doDeleteAuditRecord(method string, path string, query string, body map[string]interface{}) error {
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
