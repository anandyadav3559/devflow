package scheduler

import (
	"testing"

	internal "github.com/anandyadav3559/devflow/internal/storage"
)

func TestTopoSortDependencyOrder(t *testing.T) {
	t.Parallel()

	services := map[string]internal.Service{
		"database": {},
		"backend":  {DependsOn: []string{"database"}},
		"frontend": {DependsOn: []string{"backend"}},
	}

	order, err := TopoSort(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 services in order, got %d", len(order))
	}

	pos := make(map[string]int, len(order))
	for i, s := range order {
		pos[s] = i
	}

	if pos["database"] > pos["backend"] {
		t.Fatalf("expected database before backend, order=%v", order)
	}
	if pos["backend"] > pos["frontend"] {
		t.Fatalf("expected backend before frontend, order=%v", order)
	}
}

func TestTopoSortUnknownDependency(t *testing.T) {
	t.Parallel()

	services := map[string]internal.Service{
		"api": {DependsOn: []string{"db"}},
	}

	_, err := TopoSort(services)
	if err == nil {
		t.Fatal("expected unknown dependency error, got nil")
	}
}

func TestTopoSortCycle(t *testing.T) {
	t.Parallel()

	services := map[string]internal.Service{
		"a": {DependsOn: []string{"b"}},
		"b": {DependsOn: []string{"a"}},
	}

	_, err := TopoSort(services)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
}
