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

type addUserToGroupCmd struct {
	*baseCmd
}

func newAddUserToGroupCmd() *addUserToGroupCmd {
	ccmd := &addUserToGroupCmd{}

	cmd := &cobra.Command{
		Use:   "addUserToGroup",
		Short: "Get user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.addUserToGroup,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("tenant", "", "Tenant")
	cmd.Flags().String("groupId", "", "Group ID")
	cmd.Flags().String("userId", "", "User id")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *addUserToGroupCmd) addUserToGroup(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("userId"); err == nil && v != "" {
		if _, exists := body["userId"]; !exists {
			body["user"] = make(map[string]interface{})
		}
		body["user"].(map[string]interface{})["self"] = v
	}

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

	path := replacePathParameters("/user/{tenant}/groups/{groupId}/users", pathParameters)

	return n.doAddUserToGroup("POST", path, queryValue, body)
}

func (n *addUserToGroupCmd) doAddUserToGroup(method string, path string, query string, body map[string]interface{}) error {
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
