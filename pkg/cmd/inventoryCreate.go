package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

type newManagedObjectCmd struct {
	*baseCmd
}

func newNewManagedObjectCmd() *newManagedObjectCmd {
	ccmd := &newManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a managed object",
		Long:  `Create a managed object`,
		Example: `
			c8y inventory create --data '{\"name\":\"hello-go\"}'

			or using the json shorthand form (to create non nested data)

			c8y inventory create --data "name=hello-go,c8y_IsDevice={}"
		`,
		RunE: ccmd.newManagedObject,
	}

	// Flags
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *newManagedObjectCmd) newManagedObject(cmd *cobra.Command, args []string) error {

	data := getDataFlag(cmd)

	return n.doNewManagedObject(data)
}

func (n *newManagedObjectCmd) doNewManagedObject(data map[string]interface{}) error {
	_, resp, err := client.Inventory.Create(
		context.Background(),
		&data,
	)
	if err != nil {
		return newSystemErrorF("failed to create managed object. %s", err)
	}
	fmt.Println(*resp.JSONData)
	return nil
}
