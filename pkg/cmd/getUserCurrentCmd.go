package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getUserCurrentCmd struct {
	*baseCmd
}

func newGetUserCurrentCmd() *getUserCurrentCmd {
	ccmd := &getUserCurrentCmd{}

	cmd := &cobra.Command{
		Use:   "getCurrentUser",
		Short: "Get user",
		Long:  "",
		Example: `
        
		`,
		RunE: ccmd.getUserCurrent,
	}

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getUserCurrentCmd) getUserCurrent(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("/user/currentUser", pathParameters)

	return n.doGetUserCurrent("GET", path, queryValue, body)
}

func (n *getUserCurrentCmd) doGetUserCurrent(method string, path string, query string, body map[string]interface{}) error {
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
