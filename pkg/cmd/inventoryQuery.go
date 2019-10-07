package cmd

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

var inventoryQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query inventory managed objects",
	Long:  `Query inventory managed objects using Cumulocity's query language`,
	Run: func(cmd *cobra.Command, args []string) {

		options := &c8y.ManagedObjectOptions{
			PaginationOptions: c8y.PaginationOptions{
				PageSize:       globalFlagPageSize,
				WithTotalPages: globalFlagWithTotalPages,
			},
		}

		if val, err := cmd.Flags().GetBool(inventoryFlagWithParents); err == nil && val {
			options.WithParents = val
		}

		if val, err := cmd.Flags().GetString(inventoryFlagQuery); err == nil && val != "" {
			options.Query = val
		}

		if val, err := cmd.Flags().GetString(inventoryFlagFragmentType); err == nil && val != "" {
			options.FragmentType = val
		}

		if val, err := cmd.Flags().GetString(inventoryFlagType); err == nil && val != "" {
			options.Type = val
		}

		if val, err := cmd.Flags().GetString(inventoryFlagText); err == nil && val != "" {
			options.Text = val
		}

		_, resp, err := client.Inventory.GetManagedObjects(
			context.Background(),
			options,
		)
		if err != nil {
			panic(fmt.Errorf("%s", err))
		}
		fmt.Println(*resp.JSONData)
	},
}

func init() {
	// Flags
	inventoryQueryCmd.Flags().StringP(inventoryFlagQuery, "q", "", "name (accepts wildcards)")
	inventoryQueryCmd.Flags().String(inventoryFlagFragmentType, "", "Fragment type")
	inventoryQueryCmd.Flags().String(inventoryFlagType, "", "Type")
	inventoryQueryCmd.Flags().String(inventoryFlagText, "", "Text")
	addInventoryOptions(inventoryQueryCmd)

	inventoryQueryCmd.Flags().SetNormalizeFunc(flagNormalizeFunc)
}
