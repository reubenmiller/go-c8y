package cmd

import (
	"github.com/spf13/cobra"
)

const (
	inventoryFlagFragmentType = "fragmentType"
	inventoryFlagQuery        = "query"
	inventoryFlagType         = "type"
	inventoryFlagText         = "text"
	inventoryFlagWithParents  = "withParents"
	inventoryFlagFilter       = "filter"
	inventoryFlagID           = "id"
	inventoryFlagFile         = "file"
)

func addInventoryOptions(cmd *cobra.Command) {
	cmd.Flags().Bool(inventoryFlagWithParents, false, "With parents")
}

func addResultFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(inventoryFlagFilter, "f", "", "Filter property")
}

func addIDFlag(cmd *cobra.Command) {
	cmd.Flags().StringArrayP(inventoryFlagID, "i", []string{}, "Managed Object ID")
	cmd.MarkFlagRequired(inventoryFlagID)
}

func addApplicationFlag(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("application", "i", []string{}, "Application")
	cmd.MarkFlagRequired(inventoryFlagID)
}

func addDataFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(FlagDataName, "d", "", "json")
}

func getDataFlag(cmd *cobra.Command) map[string]interface{} {
	if value, err := cmd.Flags().GetString(FlagDataName); err == nil {
		return MustParseJSON(value)
	}
	return make(map[string]interface{})
}
