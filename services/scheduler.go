package services

import (
	"fmt"
	"sort"
)

// Start is the top-level entry point. It loads the workflow, resolves
// dependency order, and launches each service.
func Start(file string) {
	wf, err := LoadWorkflow(file)
	if err != nil {
		fmt.Println("Error loading workflow:", err)
		return
	}

	fmt.Println("Workflow Name:", wf.WorkflowName)

	order, err := topoSort(wf.Services)
	if err != nil {
		fmt.Println("Dependency error:", err)
		return
	}

	for _, name := range order {
		svc := wf.Services[name]
		if svc.Detached {
			fmt.Printf("Starting %q (detached)...\n", name)
		} else {
			fmt.Printf("Starting %q (new terminal)...\n", name)
		}

		if err := RunService(name, svc); err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
		} else {
			fmt.Printf("  ✓ OK\n")
		}
	}
}

// topoSort returns service names in dependency order using Kahn's algorithm.
// Returns an error if an unknown dependency or circular dependency is found.
func topoSort(services map[string]Service) ([]string, error) {
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dependency -> services that need it

	for name := range services {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		for _, dep := range services[name].DependsOn {
			if _, ok := services[dep]; !ok {
				return nil, fmt.Errorf("service %q depends on unknown service %q", name, dep)
			}
			dependents[dep] = append(dependents[dep], name)
			inDegree[name]++
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var ordered []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		ordered = append(ordered, node)

		deps := dependents[node]
		sort.Strings(deps)
		for _, dep := range deps {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(ordered) != len(services) {
		return nil, fmt.Errorf("circular dependency detected among services")
	}

	return ordered, nil
}
