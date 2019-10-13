package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type newAuditCmd struct {
	*baseCmd
}

func newNewAuditCmd() *newAuditCmd {
	ccmd := &newAuditCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new audit record",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.newAudit,
	}

	cmd.Flags().String("type", "", "Identifies the type of this audit record.")
	cmd.Flags().String("time", "", "Time of the audit record.")
	cmd.Flags().String("text", "", "Text description of the audit record.")
	cmd.Flags().String("source", "", "An optional ManagedObject that the audit record originated from")
	cmd.Flags().String("activity", "", "The activity that was carried out.")
	cmd.Flags().String("activity", "", "The severity of action: critical, major, minor, warning or information.")
	cmd.Flags().String("user", "", "The user responsible for the audited action.")
	cmd.Flags().String("application", "", "The application used to carry out the audited action.")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newAuditCmd) newAudit(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("time"); err == nil && v != "" {
		body["time"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil && v != "" {
		body["text"] = v
	}
	if v, err := cmd.Flags().GetString("source"); err == nil && v != "" {
		if _, exists := body["source"]; !exists {
			body["source"] = make(map[string]interface{})
		}
		body["source"].(map[string]interface{})["id"] = v
	}
	if v, err := cmd.Flags().GetString("activity"); err == nil && v != "" {
		body["activity"] = v
	}
	if v, err := cmd.Flags().GetString("activity"); err == nil && v != "" {
		body["activity"] = v
	}
	if v, err := cmd.Flags().GetString("user"); err == nil && v != "" {
		body["user"] = v
	}
	if v, err := cmd.Flags().GetString("application"); err == nil && v != "" {
		body["application"] = v
	}
	if v, err := cmd.Flags().GetString("changes"); err == nil && v != "" {
		body["changes"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/audit/auditRecords", pathParameters)

	return n.doNewAudit("POST", path, queryValue, body)
}

func (n *newAuditCmd) doNewAudit(method string, path string, query string, body map[string]interface{}) error {
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
