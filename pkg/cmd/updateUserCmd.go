package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type updateUserCmd struct {
	*baseCmd
}

func newUpdateUserCmd() *updateUserCmd {
	ccmd := &updateUserCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update user",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.updateUser,
	}

	cmd.Flags().String("firstName", "", "User first name")
	cmd.Flags().String("lastName", "", "User last name")
	cmd.Flags().String("phone", "", "User phone number. Format: '+[country code][number]', has to be a valid MSISDN")
	cmd.Flags().String("email", "", "User email address")
	cmd.Flags().Bool("enabled", false, "User activation status (true/false)")
	cmd.Flags().String("password", "", "User password. Min: 6, max: 32 characters. Only Latin1 chars allowed")
	cmd.Flags().Bool("sendPasswordResetEmail", false, "User activation status (true/false)")
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateUserCmd) updateUser(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("firstName"); err == nil && v != "" {
		body["firstName"] = v
	}
	if v, err := cmd.Flags().GetString("lastName"); err == nil && v != "" {
		body["lastName"] = v
	}
	if v, err := cmd.Flags().GetString("phone"); err == nil && v != "" {
		body["phone"] = v
	}
	if v, err := cmd.Flags().GetString("email"); err == nil && v != "" {
		body["email"] = v
	}
	if v, err := cmd.Flags().GetString("enabled"); err == nil && v != "" {
		body["enabled"] = v
	}
	if v, err := cmd.Flags().GetString("password"); err == nil && v != "" {
		body["password"] = v
	}
	if v, err := cmd.Flags().GetString("sendPasswordResetEmail"); err == nil && v != "" {
		body["sendPasswordResetEmail"] = v
	}

	// path parameters
	pathParameters := make(map[string]string)

	path := replacePathParameters("user/{tenant}/users/{id}", pathParameters)

	return n.doUpdateUser("POST", path, queryValue, body)
}

func (n *updateUserCmd) doUpdateUser(method string, path string, query string, body map[string]interface{}) error {
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
