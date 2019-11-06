// Code generated from specification version 1.0.0: DO NOT EDIT
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/howeyc/gopass"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/encrypt"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type CumulocitySessions struct {
	Sessions []CumulocitySession `json:"sessions"`
}

type CumulocitySession struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	Tenant      string `json:"tenant"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Description string `json:"description"`

	MicroserviceAliases map[string]string `json:"microserviceAliases"`
}

func (s *CumulocitySession) SetPassword(password string) {
	s.Password = encrypt.EncryptString(password, "fixed-token")
}

func (s CumulocitySession) GetPassword() string {
	return encrypt.DecryptString(s.Password, "fixed-token")
}

type newSessionCmd struct {
	*baseCmd
}

func newNewSessionCmd() *newSessionCmd {
	ccmd := &newSessionCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Cumulocity session credentials",
		Long:  `Create a new Cumulocity session credentials`,
		Example: `

		`,
		RunE: ccmd.newSession,
	}

	cmd.SilenceUsage = true

	cmd.Flags().String("host", "", "Host. .e.g. test.cumulocity.com. (required)")
	cmd.Flags().String("tenant", "", "Tenant. (required)")
	cmd.Flags().String("user", "", "User (without tenant). (required)")
	cmd.Flags().String("password", "", "Password. (required)")
	cmd.Flags().String("description", "", "Description about the session")

	// Required flags
	cmd.MarkFlagRequired("host")
	cmd.MarkFlagRequired("tenant")
	cmd.MarkFlagRequired("user")
	// cmd.MarkFlagRequired("password")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newSessionCmd) newSession(cmd *cobra.Command, args []string) error {

	session := &CumulocitySession{}

	if v, err := cmd.Flags().GetString("host"); err == nil && v != "" {
		session.Host = v
	}
	if v, err := cmd.Flags().GetString("tenant"); err == nil && v != "" {
		session.Tenant = v
	}

	if cmd.Flags().Changed("password") {
		if v, err := cmd.Flags().GetString("password"); err == nil && v != "" {
			session.SetPassword(v)
		}
	} else {
		fmt.Printf("Enter password: ")
		password, _ := gopass.GetPasswd() // Silent
		session.SetPassword(string(password))
	}
	if v, err := cmd.Flags().GetString("user"); err == nil && v != "" {
		session.Username = v
	}
	if v, err := cmd.Flags().GetString("description"); err == nil && v != "" {
		session.Description = v
	}

	log.Printf("Setting env: %s", session.Host)
	os.Setenv("C8Y_HOST_TEST", session.Host)

	log.Printf("Getting env: %s", os.Getenv("C8Y_HOST_TEST"))

	if str, err := json.Marshal(session); err == nil {
		fmt.Printf("%s", str)
	}

	return nil
	// return newUserError(fmt.Sprintf("Flag [%s] does not exist. %s", "device", err))
}

func (n *newSessionCmd) doNewSession(method string, path string, query string, body map[string]interface{}) error {
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
