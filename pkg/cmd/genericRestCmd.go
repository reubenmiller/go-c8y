package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getGenericRestCmd struct {
	*baseCmd
}

func newGetGenericRestCmd() *getGenericRestCmd {
	ccmd := &getGenericRestCmd{}

	cmd := &cobra.Command{
		Use:   "rest",
		Short: "Send generic REST request",
		Long:  `Send generic REST request`,
		Example: `
			Get a list of managed objects
			c8y rest get /alarm/alarms

			c8y rest GET "/alarm/alarms?pageSize=10&status=ACTIVE"

			// Create a new alarm
			c8y rest POST "alarm/alarms" --data "text=one,severity=MAJOR,type=test_Type,time=2019-01-01,source={'id': '12345'}"
		`,
		RunE: ccmd.getGenericRest,
	}

	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getGenericRestCmd) getGenericRest(cmd *cobra.Command, args []string) error {
	method := "get"
	var uri string
	if len(args) > 1 {
		method = args[0]
		uri = args[1]
	}

	method = strings.ToUpper(method)

	if method == "GET" || method == "DELETE" {
		return n.doGetGenericRest(method, uri)
	} else if method == "PUT" || method == "POST" {
		data := getDataFlag(cmd)
		if !cmd.Flags().Changed(FlagDataName) {
			return newUserError("Missing --data argument")
		}
		// Hide usage for system errors
		cmd.SilenceUsage = true
		return n.doDataGenericRest(method, uri, data)
	} else {
		return newUserError("Invalid method. Only GET, PUT, POST and DELETE are accepted")
	}
	return nil
}

func (n *getGenericRestCmd) doGetGenericRest(method string, path string) error {
	baseURL, _ := url.Parse(path)
	req, err := client.NewRequest(method, baseURL.Path, baseURL.RawQuery, nil)

	if err != nil {
		return newSystemError(err)
	}

	resp, err := client.Do(context.Background(), req, nil)

	if err != nil {
		return newSystemError(err)
	}

	fmt.Println(*resp.JSONData)
	return nil
}

func (n *getGenericRestCmd) doDataGenericRest(method string, path string, data map[string]interface{}) error {
	resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
			Path:         path,
			Body:         data,
			ResponseData: nil,
		})

	if err != nil {
		if resp.JSONData != nil {
			fmt.Println(*resp.JSONData)
		}
		return newSystemError(err)
	}

	fmt.Println(*resp.JSONData)
	return nil
}
