package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type queryManagedObjectCmd struct {
	*baseCmd
}

func newQueryManagedObjectCmd() *queryManagedObjectCmd {
	ccmd := &queryManagedObjectCmd{}

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query inventory managed objects",
		Long:  `Query inventory managed objects using Cumulocity's query language`,
		Example: `
			TODO
		`,
		RunE: ccmd.queryManagedObject,
	}

	// Flags
	cmd.Flags().StringP(inventoryFlagQuery, "q", "", "name (accepts wildcards)")
	cmd.Flags().String(inventoryFlagFragmentType, "", "Fragment type")
	cmd.Flags().String(inventoryFlagType, "", "Type")
	cmd.Flags().String(inventoryFlagText, "", "Text")
	addInventoryOptions(cmd)

	cmd.Flags().SetNormalizeFunc(flagNormalizeFunc)

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *queryManagedObjectCmd) queryManagedObject(cmd *cobra.Command, args []string) error {

	withParents, _ := cmd.Flags().GetBool(inventoryFlagWithParents)
	query, _ := cmd.Flags().GetString(inventoryFlagQuery)
	fragmentType, _ := cmd.Flags().GetString(inventoryFlagFragmentType)
	typeValue, _ := cmd.Flags().GetString(inventoryFlagType)
	text, _ := cmd.Flags().GetString(inventoryFlagText)

	return n.doQueryManagedObject(query, fragmentType, typeValue, text, withParents)
}

func (n *queryManagedObjectCmd) doQueryManagedObject(query, fragmentType, typeValue, text string, withParents bool) error {
	options := &c8y.ManagedObjectOptions{
		Query:        query,
		FragmentType: fragmentType,
		Type:         typeValue,
		Text:         text,
		PaginationOptions: c8y.PaginationOptions{
			PageSize:       globalFlagPageSize,
			WithTotalPages: globalFlagWithTotalPages,
		},
	}

	if withParents {
		options.WithParents = true
	}

	_, resp, err := client.Inventory.GetManagedObjects(
		context.Background(),
		options,
	)
	if err != nil {
		return errors.Wrap(err, "failed")
	}
	fmt.Println(*resp.JSONData)
	return nil
}
