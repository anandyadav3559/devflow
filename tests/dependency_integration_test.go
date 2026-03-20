package tests

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/anandyadav3559/devflow/services"
)

var workflowPath = flag.String("workflow", "workflows/workflow1.yml", "path to workflow file for integration test")

func TestWorkflowDependencies(t *testing.T) {
	// If running from tests/ directory, we might need to adjust path
	path := *workflowPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Try parent dir if running from tests/
		path = "../" + *workflowPath
	}

	wf, err := services.LoadWorkflow(path)
	if err != nil {
		t.Fatalf("Failed to load workflow from %q: %v", path, err)
	}

	fmt.Println("\n--- Dependency Graph ---")
	for name, svc := range wf.Services {
		deps := "None"
		if len(svc.DependsOn) > 0 {
			deps = strings.Join(svc.DependsOn, ", ")
		}
		
		varsStr := "{}"
		if len(svc.Vars) > 0 {
			var vParts []string
			for k, v := range svc.Vars {
				vParts = append(vParts, fmt.Sprintf("%s=%s", k, v))
			}
			varsStr = "{" + strings.Join(vParts, ", ") + "}"
		}

		fmt.Printf("[Service: %s] -> Depends On: [%s] | Args: %v | Vars: %s\n", name, deps, svc.Args, varsStr)
	}
	fmt.Println("------------------------")

	order, err := services.TopoSort(wf.Services)
	if err != nil {
		t.Fatalf("TopoSort failed: %v", err)
	}

	// Verify that dependencies come before their dependents
	position := make(map[string]int)
	for i, name := range order {
		position[name] = i
	}

	for name, svc := range wf.Services {
		for _, dep := range svc.DependsOn {
			if position[dep] >= position[name] {
				t.Errorf("service %q started at %d, but its dependency %q started later at %d", 
					name, position[name], dep, position[dep])
			}
		}
	}

	fmt.Printf("Computed execution order: %v\n", order)
}
