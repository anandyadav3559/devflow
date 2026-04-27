package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapCreatesExpectedLayout(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	if err := Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() returned error: %v", err)
	}

	base := filepath.Join(tempConfig, "devflow")
	requiredDirs := []string{
		base,
		filepath.Join(base, "storage"),
		filepath.Join(base, "logs"),
		filepath.Join(base, "flows"),
	}

	for _, dir := range requiredDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected %q to exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be directory", dir)
		}
	}

	workflowsFile := filepath.Join(base, "storage", "workflows.json")
	data, err := os.ReadFile(workflowsFile)
	if err != nil {
		t.Fatalf("expected workflows file to exist: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected workflows.json to contain [], got %q", string(data))
	}
}
