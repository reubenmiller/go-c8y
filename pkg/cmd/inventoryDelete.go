package cmd

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
)

type deleteManagedObjectCmd struct {
	*baseCmd
}

func newDeleteManagedObjectCmd() *deleteManagedObjectCmd {
	ccmd := &deleteManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete managed object/s",
		Long:  `Delete managed object/s`,
		Example: `
			Delete a list of managed objects
			c8y inventory delete --id 12345,67891
		`,
		RunE: ccmd.deleteManagedObject,
	}

	// Flags
	addIDFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *deleteManagedObjectCmd) deleteManagedObject(cmd *cobra.Command, args []string) error {
	ids := GetIDs(cmd, args)
	return n.doDeleteManagedObject(ids)
}

func (n *deleteManagedObjectCmd) doDeleteManagedObject(ids []string) error {
	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	errorsCh := make(chan error, len(ids))

	for i := range ids {
		go func(index int) {
			_, err := client.Inventory.Delete(
				context.Background(),
				ids[index],
			)

			if err != nil {
				errorsCh <- err
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	close(errorsCh)
	return newErrorSummary("command failed", errorsCh)
}
