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

type newUserCmd struct {
	*baseCmd
}

func newNewUserCmd() *newUserCmd {
	ccmd := &newUserCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user within the collection",
		Long:  ``,
		Example: `
        
		`,
		RunE: ccmd.newUser,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("userName", "", "User name, unique for a given domain. Max: 1000 characters (required)")
	cmd.Flags().String("firstName", "", "User first name")
	cmd.Flags().String("lastName", "", "User last name")
	cmd.Flags().String("phone", "", "User phone number. Format: '+[country code][number]', has to be a valid MSISDN")
	cmd.Flags().String("email", "", "User email address")
	cmd.Flags().Bool("enabled", false, "User activation status (true/false)")
	cmd.Flags().String("password", "", "User password. Min: 6, max: 32 characters. Only Latin1 chars allowed (required)")
	cmd.Flags().Bool("sendPasswordResetEmail", false, "User activation status (true/false)")
	addDataFlag(cmd)

	// Required flags
	cmd.MarkFlagRequired("userName")
	cmd.MarkFlagRequired("password")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newUserCmd) newUser(cmd *cobra.Command, args []string) error {

	// query parameters
	queryValue := url.QueryEscape("")

	// body
	var body map[string]interface{}
	body = getDataFlag(cmd)
	if v, err := cmd.Flags().GetString("userName"); err == nil && v != "" {
		body["userName"] = v
	}
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

	path := replacePathParameters("user/{tenant}/users", pathParameters)

	return n.doNewUser("POST", path, queryValue, body)
}

func (n *newUserCmd) doNewUser(method string, path string, query string, body map[string]interface{}) error {
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
