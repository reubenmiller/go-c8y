package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type getManagedObjectCmd struct {
	*baseCmd
}

func newGetManagedObjectCmd() *getManagedObjectCmd {
	ccmd := &getManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a managed object",
		Long:  `Get a managed object`,
		Example: `
			Get a list of managed objects
			c8y inventory get --id 12345 --id 67891

			Or retrieve multiple managed objects
			c8y inventory get --id 12345,67891
		`,
		RunE: ccmd.getManagedObject,
	}

	addInventoryOptions(cmd)
	addResultFilterFlags(cmd)
	addIDFlag(cmd)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *getManagedObjectCmd) getManagedObject(cmd *cobra.Command, args []string) error {

	ids := GetIDs(cmd, args)
	withParents, _ := cmd.Flags().GetBool("withParents")
	filterArg, _ := cmd.Flags().GetString("filter")
	filter := SplitString(filterArg, ",")

	return n.doGetManagedObject(ids, withParents, filter)
}

func (n *getManagedObjectCmd) doGetManagedObject(ids []string, withParents bool, filter []string) error {
	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	if len(filter) > 0 {
		// Print csv header
		fmt.Println(strings.Join(filter, ","))
	}

	errorsCh := make(chan error, len(ids))

	for i := range ids {
		go func(index int, filter []string) {
			_, resp, err := client.Inventory.GetManagedObject(
				context.Background(),
				ids[index],
				&c8y.ManagedObjectOptions{
					WithParents:       withParents,
					PaginationOptions: *c8y.NewPaginationOptions(1), // TODO: This should not be required as it is not supported by the api!
				},
			)

			if err != nil {
				errorsCh <- err
			} else {
				if len(filter) > 0 {
					selectedOutput := FilterJSON(*resp.JSON, filter)
					fmt.Println(strings.Join(selectedOutput, ","))
				} else {
					fmt.Println(*resp.JSONData)
				}
			}
			wg.Done()
		}(i, filter)
	}

	wg.Wait()
	close(errorsCh)

	var errorSummary error
	for err := range errorsCh {
		if err != nil {
			if errorSummary == nil {
				errorSummary = errors.New("command failed")
			}
			errorSummary = errors.WithStack(err)
		}
	}

	return errorSummary
}
