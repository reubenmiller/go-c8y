package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getGroupReferenceCollectionCmd struct {
	*baseCmd
}

func newGetGroupReferenceCollectionCmd() *getGroupReferenceCollectionCmd {
	ccmd := &getGroupReferenceCollectionCmd{}

	cmd := &cobra.Command{
		Use:   "getCollection",
		Short: "Get information about all groups of a user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.getGroupReferenceCollection,
	}

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("username", "", "Username")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGroupReferenceCollectionCmd) getGroupReferenceCollection(cmd *cobra.Command, args []string) error {

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

	path := replacePathParameters("/user/{tenant}/users/{username}/groups", pathParameters)

	return n.doGetGroupReferenceCollection("GET", path, queryValue, body)
}

func (n *getGroupReferenceCollectionCmd) doGetGroupReferenceCollection(method string, path string, query string, body map[string]interface{}) error {
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
