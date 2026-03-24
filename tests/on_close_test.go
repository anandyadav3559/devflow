package tests

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	internal "github.com/anandyadav3559/devflow/internal/storage"
	"github.com/anandyadav3559/devflow/services/scheduler"
)

func TestOnClose(t *testing.T) {
	// Create a dummy workflow
	wf := internal.Workflow{
		WorkflowName: "Test OnClose",
		Services: map[string]internal.Service{
			"svc1": {
				Command: "echo",
				Args:    []string{"svc1"},
				OnClose: internal.CleanupCommands{
					{Command: "echo", Args: []string{"cleanup-svc1"}},
				},
			},
			"svc2": {
				Command:   "echo",
				Args:      []string{"svc2"},
				DependsOn: []string{"svc1"},
				OnClose: internal.CleanupCommands{
					{Command: "echo", Args: []string{"cleanup-svc2"}},
				},
			},
		},
		OnClose: internal.CleanupCommands{
			{Command: "echo", Args: []string{"cleanup-global"}},
		},
	}

	order := []string{"svc1", "svc2"}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run cleanup directly as testing the full Start with signals is complex and brittle
	// We want to verify the order and execution of cleanup commands.
	scheduler.RunCleanup(&wf, order, nil, nil, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify the order of execution
	expectedOrder := []string{
		"Cleaning up service: \"svc2\"",
		"→ Executing: echo [cleanup-svc2]",
		"Cleaning up service: \"svc1\"",
		"→ Executing: echo [cleanup-svc1]",
		"Running global cleanup...",
		"→ Executing: echo [cleanup-global]",
	}

	for _, expected := range expectedOrder {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}

	// Check strict ordering by finding indices
	idxSvc2 := strings.Index(output, "cleanup-svc2")
	idxSvc1 := strings.Index(output, "cleanup-svc1")
	idxGlobal := strings.Index(output, "cleanup-global")

	if idxSvc2 == -1 || idxSvc1 == -1 || idxGlobal == -1 {
		t.Fatalf("Missing output strings. Output: %s", output)
	}

	if !(idxSvc2 < idxSvc1 && idxSvc1 < idxGlobal) {
		t.Errorf("Cleanup executed in wrong order. Expected svc2 -> svc1 -> global. Output:\n%s", output)
	}
}
