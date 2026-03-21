package scheduler

import (
	"fmt"
	"sort"

	"github.com/anandyadav3559/devflow/services"
)

// TopoSort returns service names in dependency order using Kahn's algorithm.
// Returns an error if an unknown dependency or circular dependency is found.
func TopoSort(services map[string]services.Service) ([]string, error) {
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
