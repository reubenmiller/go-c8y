package cmd

import (
    "context"
    "fmt"
    "net/url"

    "github.com/reubenmiller/go-c8y/pkg/c8y"
    "github.com/spf13/cobra"
)

type Cmd struct {
    *baseCmd
}

func newCmd() *Cmd {
	ccmd := &Cmd{}

	cmd := &cobra.Command{
		Use:   "",
		Short: "",
		Long:  ``,
        Example: `
        
		`,
		RunE: ccmd.,
	}

    

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *Cmd) (cmd *cobra.Command, args []string) error {

    // query parameters
    queryValue := url.QueryEscape("")
    

    // body
    var body map[string]interface{}
    

    // path parameters
    pathParameters := make(map[string]string)
    
    path := replacePathParameters("", pathParameters)

    return n.do("", path, queryValue, body)
}

func (n *Cmd) do(method string, path string, query string, body map[string]interface{}) error {
    resp, err := client.SendRequest(
		context.Background(),
		c8y.RequestOptions{
			Method:       method,
            Path:         path,
            Query:        query,
			Body:         body,
		})

    if resp != nil && resp.JSONData != nil {
        fmt.Println(*resp.JSONData)
    }
	if err != nil {
		return newSystemError("command failed", err)
	}
	return nil
}
