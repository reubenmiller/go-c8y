// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/fatih/color"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/mapbuilder"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type deleteAuditRecordCmd struct {
	*baseCmd
}

func newDeleteAuditRecordCmd() *deleteAuditRecordCmd {
	ccmd := &deleteAuditRecordCmd{}

	cmd := &cobra.Command{
		Use:   "deleteCollection",
		Short: "Delete a collection of audit records",
		Long:  `Important: This method has been deprecated and will be removed completely with the July 2020 release (10.6.6). With Cumulocity IoT >= 10.6.6 the deletion of audit logs will no longer be permitted. All DELETE requests to the audit API will return the error 405 Method not allowed. Note that retention rules still apply to audit logs and will delete audit log records older than the specified retention time.`,
		Example: `

		`,
		RunE: ccmd.deleteAuditRecord,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("source", "", "Source Id or object containing an .id property of the element that should be detected. i.e. AlarmID, or Operation ID. Note: Only one source can be provided")
	cmd.Flags().String("type", "", "Type")
	cmd.Flags().String("user", "", "Username")
	cmd.Flags().String("application", "", "Application")
	cmd.Flags().String("dateFrom", "", "Start date or date and time of audit record occurrence.")
	cmd.Flags().String("dateTo", "", "End date or date and time of audit record occurrence.")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteAuditRecordCmd) deleteAuditRecord(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")
	query := url.Values{}
	if cmd.Flags().Changed("pageSize") {
		if v, err := cmd.Flags().GetInt("pageSize"); err == nil && v > 0 {
			query.Add("pageSize", fmt.Sprintf("%d", v))
		}
	}

	if cmd.Flags().Changed("withTotalPages") {
		if v, err := cmd.Flags().GetBool("withTotalPages"); err == nil && v {
			query.Add("withTotalPages", "true")
		}
	}
	queryValue, err := url.QueryUnescape(query.Encode())

	if err != nil {
		return newSystemError("Invalid query parameter")
	}

	// form data
	formData := make(map[string]io.Reader)

	// body
	body := mapbuilder.NewMapBuilder()

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("source"); err == nil {
		if v != "" {
			pathParameters["source"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "source", err))
	}
	if v, err := cmd.Flags().GetString("type"); err == nil {
		if v != "" {
			pathParameters["type"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "type", err))
	}
	if v, err := cmd.Flags().GetString("user"); err == nil {
		if v != "" {
			pathParameters["user"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "user", err))
	}
	if v, err := cmd.Flags().GetString("application"); err == nil {
		if v != "" {
			pathParameters["application"] = v
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "application", err))
	}
	if cmd.Flags().Changed("dateFrom") {
		if v, err := tryGetTimestampFlag(cmd, "dateFrom"); err == nil && v != "" {
			pathParameters["dateFrom"] = v
		} else {
			return newUserError("invalid date format", err)
		}
	}
	if cmd.Flags().Changed("dateTo") {
		if v, err := tryGetTimestampFlag(cmd, "dateTo"); err == nil && v != "" {
			pathParameters["dateTo"] = v
		} else {
			return newUserError("invalid date format", err)
		}
	}

	path := replacePathParameters("/audit/auditRecords", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	req := c8y.RequestOptions{
		Method:       "DELETE",
		Path:         path,
		Query:        queryValue,
		Body:         body.GetMap(),
		FormData:     formData,
		IgnoreAccept: false,
		DryRun:       globalFlagDryRun,
	}

	return n.doDeleteAuditRecord(req, filters)
}

func (n *deleteAuditRecordCmd) doDeleteAuditRecord(req c8y.RequestOptions, filters *JSONFilters) error {
	resp, err := client.SendRequest(
		context.Background(),
		req,
	)

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		// estimate size based on utf8 encoding. 1 char is 1 byte
		Logger.Printf("Response Length: %0.1fKB", float64(len(*resp.JSONData)*1)/1024)

		var responseText []byte

		if filters != nil && !globalFlagRaw {
			responseText = filters.Apply(*resp.JSONData, "")
		} else {
			responseText = []byte(*resp.JSONData)
		}

		if globalFlagPrettyPrint && json.Valid(responseText) {
			fmt.Printf("%s", pretty.Pretty(responseText))
		} else {
			fmt.Printf("%s", responseText)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
