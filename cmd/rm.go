package cmd

import (
	"fmt"

	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm [workflow_name]",
	Short:   "Remove a registered workflow",
	Long:    `Deletes a workflow from devflow's registry and removes its snapshotted copy in .devflow/flows.`,
	Example: `  devflow rm my_custom_workflow`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := storage.Bootstrap(); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		if err := storage.DeleteWorkflow(name); err != nil {
			return err
		}

		fmt.Printf("Workflow %q deleted successfully.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
