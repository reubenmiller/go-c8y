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

type newManagedObjectCmd struct {
	*baseCmd
}

func newNewManagedObjectCmd() *newManagedObjectCmd {
	ccmd := &newManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new inventory",
		Long:  `Create a new inventory`,
		Example: `
        
		`,
		RunE: ccmd.newManagedObject,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "The ManagedObject that the inventory originated from (required)")
	cmd.Flags().String("type", "", "Identifies the type of this inventory, e.g. 'com_cumulocity_events_TamperEvent'. (required)")
	cmd.Flags().String("time", "", "Time of the inventory. (required)")
	cmd.Flags().String("text", "", "Text description of the inventory. (required)")
	cmd.Flags().String("severity", "", "The severity of the inventory: CRITICAL, MAJOR, MINOR or WARNING. Must be upper-case. (required)")
	cmd.Flags().String("status", "", "The status of the inventory: ACTIVE, ACKNOWLEDGED or CLEARED. If status was not appeared, new inventory will have status ACTIVE. Must be upper-case.")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("device")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("time")
	cmd.MarkFlagRequired("text")
	cmd.MarkFlagRequired("severity")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newManagedObjectCmd) newManagedObject(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetStringSlice("device"); err == nil {
		for _, iValue := range v {
			if _, exists := body["source"]; !exists {
				body["source"] = make(map[string]interface{})
			}
			body["source"].(map[string]interface{})["id"] = iValue
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "device", err))
	}
	if v, err := cmd.Flags().GetString("type"); err == nil && v != "" {
		body["type"] = v
	}
	if v, err := tryGetTimestampFlag(cmd, "time"); err == nil && v != "" {
		body["time"] = v
	}
	if v, err := cmd.Flags().GetString("text"); err == nil && v != "" {
		body["text"] = v
	}
	if v, err := cmd.Flags().GetString("severity"); err == nil && v != "" {
		body["severity"] = v
	}
	if v, err := cmd.Flags().GetString("status"); err == nil && v != "" {
		body["status"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("inventory/managedObjects", pathParameters)

	return n.doNewManagedObject("POST", path, queryValue, body)
}

func (n *newManagedObjectCmd) doNewManagedObject(method string, path string, query string, body map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Query:        query,
			Body:         body,
			IgnoreAccept: false,
			DryRun:       globalFlagDryRun,
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
