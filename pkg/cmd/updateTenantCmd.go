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

type updateTenantCmd struct {
	*baseCmd
}

func newUpdateTenantCmd() *updateTenantCmd {
	ccmd := &updateTenantCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update tenant",
		Long:  ``,
		Example: `

		`,
		RunE: ccmd.updateTenant,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant id")
	cmd.Flags().String("company", "", "Company name. Maximum 256 characters (required)")
	cmd.Flags().String("domain", "", "Domain name to be used for the tenant. Maximum 256 characters (required)")
	cmd.Flags().String("adminName", "", "Username of the tenant administrator")
	cmd.Flags().String("adminPass", "", "Password of the tenant administrator")
	cmd.Flags().String("contactName", "", "A contact name, for example an administrator, of the tenant")
	cmd.Flags().String("contact_phone", "", "An international contact phone number")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("company")
	cmd.MarkFlagRequired("domain")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateTenantCmd) updateTenant(cmd *cobra.Command, args []string) error {

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
	body.SetMap(getDataFlag(cmd))
	if v, err := cmd.Flags().GetString("company"); err == nil {
		if v != "" {
			body.Set("company", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "company", err))
	}
	if v, err := cmd.Flags().GetString("domain"); err == nil {
		if v != "" {
			body.Set("domain", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "domain", err))
	}
	if v, err := cmd.Flags().GetString("adminName"); err == nil {
		if v != "" {
			body.Set("adminName", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "adminName", err))
	}
	if v, err := cmd.Flags().GetString("adminPass"); err == nil {
		if v != "" {
			body.Set("adminPass", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "adminPass", err))
	}
	if v, err := cmd.Flags().GetString("contactName"); err == nil {
		if v != "" {
			body.Set("contactName", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "contactName", err))
	}
	if v, err := cmd.Flags().GetString("contact_phone"); err == nil {
		if v != "" {
			body.Set("contact_phone", v)
		}
	} else {
		return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "contact_phone", err))
	}

	// path parameters
	pathParameters := make(map[string]string)
	if v := getTenantWithDefaultFlag(cmd, "tenant", client.TenantName); v != "" {
		pathParameters["tenant"] = v
	}

	path := replacePathParameters("/tenant/tenants/{tenant}", pathParameters)

	// filter and selectors
	filters := getFilterFlag(cmd, "filter")

	req := c8y.RequestOptions{
		Method:       "PUT",
		Path:         path,
		Query:        queryValue,
		Body:         body.GetMap(),
		FormData:     formData,
		IgnoreAccept: false,
		DryRun:       globalFlagDryRun,
	}

	return n.doUpdateTenant(req, filters)
}

func (n *updateTenantCmd) doUpdateTenant(req c8y.RequestOptions, filters *JSONFilters) error {
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
