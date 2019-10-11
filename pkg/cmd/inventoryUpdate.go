package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type updateManagedObjectCmd struct {
	*baseCmd
}

func newUpdateManagedObjectCmd() *updateManagedObjectCmd {
	ccmd := &updateManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a managed object",
		Long:  `Update a managed object`,
		Example: `
			c8y inventory update --id 12345 --data '{\"name\":\"hello-go\"}'

			or using the json shorthand form (to create non nested data)

			c8y inventory update --data "name=hello-go,c8y_IsDevice={}"
		`,
		RunE: ccmd.updateManagedObject,
	}

	// Flags
	addIDFlag(cmd)
	addDataFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *updateManagedObjectCmd) updateManagedObject(cmd *cobra.Command, args []string) error {
	ids := GetIDs(cmd, args)
	data := getDataFlag(cmd)
	return n.doUpdateManagedObject(ids, data)
}

func (n *updateManagedObjectCmd) doUpdateManagedObject(ids []string, data map[string]interface{}) error {
	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	errorsCh := make(chan error, len(ids))

	for i := range ids {
		go func(index int) {
			_, resp, err := client.Inventory.Update(
				context.Background(),
				ids[index],
				data,
			)
			if err != nil {
				errorsCh <- err
			} else {
				fmt.Println(*resp.JSONData)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(errorsCh)
	return newErrorSummary("command failed", errorsCh)
}
