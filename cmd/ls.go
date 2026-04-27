package cmd

import (
	"fmt"

	"github.com/anandyadav3559/devflow/internal/config"
	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all registered workflows",
	Long:    `Lists all workflows that have been previously built and saved `,
	Example: `  devflow ls`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// load config
		if err := config.Load(); err != nil {
			fmt.Println("Warning:", err)
		}

		// ensure dirs exist
		if err := storage.Bootstrap(); err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		// load workflows
		wfs, err := storage.LoadWorkflows()
		if err != nil {
			return fmt.Errorf("error loading workflows: %w", err)
		}

		if len(wfs) == 0 {
			fmt.Println("No workflows registered yet. Use 'devflow build -f <file>' to register one.")
			return nil
		}

		fmt.Println("Registered workflows:")
		for _, wf := range wfs {
			fmt.Printf("  - %s (ID: %s)\n", wf.Name, wf.UID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
