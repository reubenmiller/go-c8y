package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getRoleReferenceCollectionFromGroupCmd struct {
	*baseCmd
}

func newGetRoleReferenceCollectionFromGroupCmd() *getRoleReferenceCollectionFromGroupCmd {
	ccmd := &getRoleReferenceCollectionFromGroupCmd{}

	cmd := &cobra.Command{
		Use:   "getRoleReferenceCollectionFromGroup",
		Short: "Get collection of user role references from a group",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getRoleReferenceCollectionFromGroup,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getRoleReferenceCollectionFromGroupCmd) getRoleReferenceCollectionFromGroup(cmd *cobra.Command, args []string) error {

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
	if v, err := cmd.Flags().GetString("groupId"); err == nil {
		pathParameters["groupId"] = v
	} else {
		return newUserError("Flag does not exist")
	}

	path := replacePathParameters("/user/{tenant}/groups/{groupId}/roles", pathParameters)

	return n.doGetRoleReferenceCollectionFromGroup("GET", path, queryValue, body)
}

func (n *getRoleReferenceCollectionFromGroupCmd) doGetRoleReferenceCollectionFromGroup(method string, path string, query string, body map[string]interface{}) error {
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
