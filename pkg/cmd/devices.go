package cmd

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "Get a list of devices",
	Long:  `All devices`,
	Run: func(cmd *cobra.Command, args []string) {

		query := fmt.Sprintf("has(c8y_IsDevice) and name eq '%s'", devicesArgName)

		if devicesArgType != "" {
			query = fmt.Sprintf("%s and type eq '%s'", query, devicesArgType)
		}

		_, resp, err := client.Inventory.GetManagedObjects(
			context.Background(),
			&c8y.ManagedObjectOptions{
				Query: query,
				PaginationOptions: c8y.PaginationOptions{
					PageSize:       globalFlagPageSize,
					WithTotalPages: globalFlagWithTotalPages,
				},
			},
		)
		if err != nil {
			panic(fmt.Errorf("%s", err))
		}
		fmt.Println(*resp.JSONData)
	},
}

var (
	devicesArgName string
	devicesArgType string
)

func init() {
	rootCmd.AddCommand(devicesCmd)

	// Flags
	devicesCmd.Flags().StringVarP(&devicesArgName, "name", "n", "*", "name (accepts wildcards")
	devicesCmd.Flags().StringVarP(&devicesArgType, "type", "t", "", "type")
}
