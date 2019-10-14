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

type getRoleReferenceCollectionFromUserCmd struct {
	*baseCmd
}

func newGetRoleReferenceCollectionFromUserCmd() *getRoleReferenceCollectionFromUserCmd {
	ccmd := &getRoleReferenceCollectionFromUserCmd{}

	cmd := &cobra.Command{
		Use:   "getRoleReferenceCollectionFromUser",
		Short: "Get collection of user role references from a user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getRoleReferenceCollectionFromUser,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "Username (required)")

	// Required flags
	cmd.MarkFlagRequired("username")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getRoleReferenceCollectionFromUserCmd) getRoleReferenceCollectionFromUser(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)
	if v, err := cmd.Flags().GetString("tenant"); err == nil {
		pathParameters["tenant"] = v
	} else {
		return newUserError("Flag does not exist")
	}
	if v, err := cmd.Flags().GetString("username"); err == nil {
		pathParameters["username"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/users/{username}/roles", pathParameters)

	return n.doGetRoleReferenceCollectionFromUser("GET", path, queryValue, body)
}

func (n *getRoleReferenceCollectionFromUserCmd) doGetRoleReferenceCollectionFromUser(method string, path string, query string, body map[string]interface{}) error {
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
