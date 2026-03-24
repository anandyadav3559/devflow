package cmd

import (
	"fmt"
	"os"

	"github.com/anandyadav3559/devflow/internal/config"
	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all registered workflows",
	Long:    `Lists all workflows that have been previously built and saved `,
	Example: `  devflow ls`,
	Run: func(cmd *cobra.Command, args []string) {
		// load config
		if err := config.Load(); err != nil {
			fmt.Println("Warning:", err)
		}

		// ensure dirs exist
		storage.Bootstrap()

		// load workflows
		wfs, err := storage.LoadWorkflows()
		if err != nil {
			fmt.Printf("Error loading workflows: %v\n", err)
			os.Exit(1)
		}

		if len(wfs) == 0 {
			fmt.Println("No workflows registered yet. Use 'devflow build -f <file>' to register one.")
			return
		}

		fmt.Println("Registered workflows:")
		for _, wf := range wfs {
			fmt.Printf("  - %s (ID: %s)\n", wf.Name, wf.UID)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
