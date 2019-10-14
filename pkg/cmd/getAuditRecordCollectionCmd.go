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

type getAuditRecordCollectionCmd struct {
	*baseCmd
}

func newGetAuditRecordCollectionCmd() *getAuditRecordCollectionCmd {
	ccmd := &getAuditRecordCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getCollection",
		Short: "Get collection of (user) audits",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getAuditRecordCollection,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("source", "", "Source Id or object containing an .id property of the element that should be detected. i.e. AlarmID, or Operation ID. Note: Only one source can be provided")
	cmd.Flags().String("type", "", "Type")
	cmd.Flags().String("user", "", "Username")
	cmd.Flags().String("application", "", "Application")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of audit record occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of audit record occurrence.")
	cmd.Flags().Bool("revert", false, "Return the newest instead of the oldest audit records. Must be used with dateFrom and dateTo parameters")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getAuditRecordCollectionCmd) getAuditRecordCollection(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if v, err := cmd.Flags().GetString("source"); err == nil {
		if v != "" {
			query.Add("source", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			query.Add("type", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("user"); err == nil {
		if v != "" {
			query.Add("user", url.QueryEscape(v))
		}
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("application"); err == nil {
		if v != "" {
			query.Add("application", url.QueryEscape(v))
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
	if v, err := cmd.Flags().GetBool("revert"); err == nil {
		if v {
			query.Add("revert", "true")
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

	path := replacePathParameters("/audit/auditRecords", pathParameters)

	return n.doGetAuditRecordCollection("GET", path, queryValue, body)
}

func (n *getAuditRecordCollectionCmd) doGetAuditRecordCollection(method string, path string, query string, body map[string]interface{}) error {
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
