// Code generated from specification version 1.0.0: DO NOT EDIT
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

type updateRetentionRuleCmd struct {
	*baseCmd
}

func newUpdateRetentionRuleCmd() *updateRetentionRuleCmd {
	ccmd := &updateRetentionRuleCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update retention rule",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateRetentionRule,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("id", "", "Retention rule id (required)")
	cmd.Flags().String("dataType", "", "RetentionRule will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *]. (required)")
	cmd.Flags().String("fragmentType", "", "RetentionRule will be applied to documents with fragmentType.")
	cmd.Flags().String("type", "", "RetentionRule will be applied to documents with type.")
	cmd.Flags().String("source", "", "RetentionRule will be applied to documents with source.")
	cmd.Flags().Bool("editable", false, "Whether the rule is editable. Can be updated only by management tenant.")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("dataType")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateRetentionRuleCmd) updateRetentionRule(cmd *cobra.Command, args []string) error {

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

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("dataType"); err == nil && v != "" {
		body["dataType"] = v
	}
	if v, err := cmd.Flags().GetString("fragmentType"); err == nil && v != "" {
		body["fragmentType"] = v
	}
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := cmd.Flags().GetString("source"); err == nil && v != "" {
		body["source"] = v
	}
	if v, err := cmd.Flags().GetString("maximumAge"); err == nil && v != "" {
		body["maximumAge"] = v
	}
	if v, err := cmd.Flags().GetString("editable"); err == nil && v != "" {
		body["editable"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("id"); err == nil {
		pathParameters["id"] = v
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "id", err))
	}

	path := replacePathParameters("/retention/retentions/{id}", pathParameters)

	return n.doUpdateRetentionRule("PUT", path, queryValue, body)
}

func (n *updateRetentionRuleCmd) doUpdateRetentionRule(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
		})

	if err != nil {
		color.Set(color.FgRed, color.Bold)
	}

	if resp != nil && resp.JSONData != nil {
		if globalFlagPrettyPrint {
			fmt.Printf("%s\n", pretty.Pretty([]byte(*resp.JSONData)))
		} else {
			fmt.Printf("%s\n", *resp.JSONData)
		}
	}

	color.Unset()

	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
