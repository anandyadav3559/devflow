package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anandyadav3559/devflow/internal/config"
	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services"
	"github.com/anandyadav3559/devflow/services/scheduler"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var buildFile string
var buildName string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Register a workflow for reuse",
	Long: `Parses your YAML workflow file to ensure it's syntactically correct 
and that there are no circular dependencies. If the workflow is valid, it saves 
a reference in your local DevFlow storage (.devflow/storage) with a unique ID 
so that it can be tracked and run easily in the future.`,
	Example: `  devflow build -f workflows/my-project.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		// load config
		if err := config.Load(); err != nil {
			fmt.Println("Warning:", err)
		}

		// ensure dirs exist
		storage.Bootstrap()

		// validate file
		if _, err := os.Stat(buildFile); os.IsNotExist(err) {
			fmt.Printf("Error: Workflow file %q not found\n", buildFile)
			os.Exit(1)
		}

		// Resolve absolute path for storage
		absPath, err := filepath.Abs(buildFile)
		if err != nil {
			absPath = buildFile
		}

		wf, err := services.LoadWorkflow(absPath)
		if err != nil {
			fmt.Printf("Error loading workflow: %v\n", err)
			os.Exit(1)
		}

		if err := validateworkflow(wf); err != nil {
			fmt.Printf("Validation failed: %v\n", err)
			os.Exit(1)
		}

		finalName := wf.WorkflowName
		if buildName != "" {
			finalName = buildName
		}

		// Check for conflicts before saving
		wfs, err := storage.LoadWorkflows()
		if err != nil {
			fmt.Printf("Error checking existing workflows: %v\n", err)
			os.Exit(1)
		}

		nameExists := func(name string) bool {
			for _, existing := range wfs {
				if existing.Name == name && existing.File != absPath {
					return true
				}
			}
			return false
		}

		if nameExists(finalName) {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("A workflow named %q already exists from a different file.\n", finalName)
			for {
				fmt.Print("Enter a new, unique name for this workflow: ")
				inputName, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("\nAborted.")
					os.Exit(1)
				}
				
				inputName = strings.TrimSpace(inputName)
				if inputName != "" {
					if !nameExists(inputName) {
						finalName = inputName
						break
					}
					fmt.Printf("Name %q also exists.\n", inputName)
				}
			}
		}

		// Register workflow in storage
		metadata := storage.WorkflowMetadata{
			UID:  storage.GenerateUID(),
			Name: finalName,
			File: absPath,
		}

		if err := storage.SaveWorkflow(metadata); err != nil {
			fmt.Printf("Error saving workflow reference: %v\n", err)
			os.Exit(1)
		}

		// Update name internally to match the new unique assignment + UID
		// This ensures when you run this snapshot later, the logs output accurately
		wf.WorkflowName = fmt.Sprintf("%s_%s", finalName, metadata.UID)

		// Save a local snapshot copy into .devflow/flows
		updatedContent, err := yaml.Marshal(wf)
		if err == nil {
			flowPath := filepath.Join(storage.GetFlowsPath(), finalName+".yml")
			os.WriteFile(flowPath, updatedContent, 0644)
		}

		fmt.Printf("Workflow %q successfully built and registered with ID: %s\n", metadata.Name, metadata.UID)
	},
}

func init() {
	buildCmd.Flags().StringVarP(&buildFile, "file", "f", "", "Path to workflow file")
	buildCmd.Flags().StringVarP(&buildName, "name", "n", "", "Custom name for the registered workflow")
	buildCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(buildCmd)
}

func validateworkflow(workflow *storage.Workflow) error {
	if workflow.WorkflowName == "" {
		return fmt.Errorf("workflow_name is required")
	}

	if len(workflow.Services) == 0 {
		return fmt.Errorf("workflow must define at least one service in 'services'")
	}

	for name, svc := range workflow.Services {
		if svc.Command == "" {
			return fmt.Errorf("service %q is missing a required 'command'", name)
		}
	}

	// Use TopoSort to check for circular dependencies or unknown dependencies reference
	_, err := scheduler.TopoSort(workflow.Services)
	if err != nil {
		return fmt.Errorf("dependency error: %v", err)
	}

	return nil
}