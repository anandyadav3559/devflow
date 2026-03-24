package main

import (
	"github.com/anandyadav3559/devflow/cmd"
)

func main() {
	cmd.Execute()
}

/*

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/anandyadav3559/devflow/services/scheduler"
)

func main() {
	workflowFile := flag.String("f", "workflows/workflow.yml", "Path to the workflow YAML file")
	flag.Parse()

	if _, err := os.Stat(*workflowFile); os.IsNotExist(err) {
		fmt.Printf("Error: Workflow file %q not found.\n", *workflowFile)
		os.Exit(1)
	}

	scheduler.Start(*workflowFile)
}
*/
