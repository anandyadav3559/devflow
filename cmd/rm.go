package cmd

import (
	"fmt"
	"os"

	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:     "rm [workflow_name]",
	Short:   "Remove a registered workflow",
	Long:    `Deletes a workflow from devflow's registry and removes its snapshotted copy in .devflow/flows.`,
	Example: `  devflow rm my_custom_workflow`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		
		storage.Bootstrap()

		if err := storage.DeleteWorkflow(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Workflow %q deleted successfully.\n", name)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
