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

	cmd.Flags().String("id", "", "Retention rule id")
	cmd.Flags().String("dataType", "", "RetentionRule will be applied to this type of documents, possible values [ALARM, AUDIT, EVENT, MEASUREMENT, OPERATION, *].")
	cmd.Flags().String("fragmentType", "", "RetentionRule will be applied to documents with fragmentType.")
	cmd.Flags().String("type", "", "RetentionRule will be applied to documents with type.")
	cmd.Flags().String("source", "", "RetentionRule will be applied to documents with source.")
	cmd.Flags().Bool("editable", false, "Whether the rule is editable. Can be updated only by management tenant.")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateRetentionRuleCmd) updateRetentionRule(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

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
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/retention/retentions/{id}", pathParameters)

	return n.doUpdateRetentionRule("PUT", path, queryValue, body)
}

func (n *updateRetentionRuleCmd) doUpdateRetentionRule(method string, path string, query string, body map[string]interface{}) error {
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
