package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anandyadav3559/devflow/internal/config"
	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services/scheduler"
	"github.com/spf13/cobra"
)

var workflowFile string
var startName string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a workflow (by name or file)",
	Long: `Starts a workflow defined in the specified YAML file or by its registered name. 
This command resolves dependencies among services and safely launches them 
in individual terminal windows (or detached processes). When you kill the session, 
DevFlow automatically runs the cleanup commands for each service.`,
	Example: `  devflow start -f workflows/my-project.yml
  devflow start -n my_custom_workflow`,
	Run: func(cmd *cobra.Command, args []string) {

		// load config
		if err := config.Load(); err != nil {
			fmt.Println("Warning:", err)
		}

		// ensure dirs exist (lazy init)
		storage.Bootstrap()

		// resolve by name if specified
		if startName != "" {
			workflowFile = filepath.Join(storage.GetFlowsPath(), startName+".yml")
		} else if workflowFile == "" {
			fmt.Println("Error: You must specify either --file (-f) or --name (-n)")
			os.Exit(1)
		}

		// validate file
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			fmt.Printf("Error: Workflow file %q not found\n", workflowFile)
			os.Exit(1)
		}

		// start workflow
		scheduler.Start(workflowFile)
	},
}

func init() {
	startCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Path to workflow file")
	startCmd.Flags().StringVarP(&startName, "name", "n", "", "Name of a registered workflow to start")
	
	rootCmd.AddCommand(startCmd)
}
